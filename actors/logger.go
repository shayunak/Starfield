package actors

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
)

type Logger struct {
	ConsellationName    string
	TimeStep            int
	TimeStamp           int
	TotalSimulationTime int // in seconds
	// Simulation Mode
	LoggerDeviceChannels        *LoggerDeviceChannels
	RemainingUnprocessedPackets int
	DeviceNames                 []string
	Events                      helpers.SimulationEntryList
	// Distances Mode
	DistancesLoggerChannels *DistanceLoggerDeviceChannels
	DistanceEntries         helpers.DistanceEntryList
}

type DistanceLoggerDeviceChannel chan UpdateDistancesMessage
type DistanceLoggerDeviceChannels []*DistanceLoggerDeviceChannel

type ILogger interface {
	// Distances Mode
	RunDistances(wg *sync.WaitGroup)
	GetDistancesDeviceChannels() *DistanceLoggerDeviceChannels
	SetDistancesDeviceChannels(channels *DistanceLoggerDeviceChannels)
	GetDistancesNumberOfDevices() int
	addNewDistanceEntries(distancesMessage UpdateDistancesMessage)
	addNewDistanceEntry(entry *helpers.DistanceEntry)
	logDistancesSimulationSummary()
	// Simulation Mode
	SetDeviceChannels(channels *LoggerDeviceChannels, deviceNames []string)
	GetNumberOfDevices() int
	GetDeviceChannels() *LoggerDeviceChannels
	GetDeviceNames() []string
	GetRemainingUnprocessedPackets() int
	ProcessEvent(event SimulationEvent, sourceIndex int)
	CloseChannels()
	logSimulationSummary()
	Run(wg *sync.WaitGroup)
	// General
	GetTotalSimulationTime() int
}

func (logger *Logger) GetTotalSimulationTime() int {
	return logger.TotalSimulationTime
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

type UpdateDistancesMessage struct {
	DeviceName string
	TimeStamp  int
	Distances  map[string]float64
}

func initDistancesChannelCases(selectCases *[]reflect.SelectCase, logger ILogger) {
	channels := *logger.GetDistancesDeviceChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func deleteDistancesDevice(logger ILogger, index int) {
	devices := *logger.GetDistancesDeviceChannels()
	devices = append(devices[:index], devices[index+1:]...)
	logger.SetDistancesDeviceChannels(&devices)
}

func startDistancesLogger(logger ILogger, wg *sync.WaitGroup) {
	for logger.GetDistancesNumberOfDevices() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, logger.GetDistancesNumberOfDevices())
		initDistancesChannelCases(&selectSatellitesCases, logger)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			deleteDistancesDevice(logger, chosen)
		}
		distanceUpdateMessage := value.Interface().(UpdateDistancesMessage)
		logger.addNewDistanceEntries(distanceUpdateMessage)
	}
	logger.logDistancesSimulationSummary()
	wg.Done()
}

func (logger *Logger) RunDistances(wg *sync.WaitGroup) {
	log.Default().Println("Running Distance Analyzer...")
	go startDistancesLogger(logger, wg)
}

func (logger *Logger) addNewDistanceEntries(distancesMessage UpdateDistancesMessage) {
	devices := distancesMessage.Distances
	for deviceId, distance := range devices {
		logger.addNewDistanceEntry(&helpers.DistanceEntry{
			TimeStamp:  distancesMessage.TimeStamp,
			FromDevice: distancesMessage.DeviceName,
			ToDevice:   deviceId,
			Distance:   int(distance),
		})
	}
	if logger.TimeStamp < distancesMessage.TimeStamp {
		logger.TimeStamp = distancesMessage.TimeStamp
	}
}

func (logger *Logger) GetDistancesDeviceChannels() *DistanceLoggerDeviceChannels {
	return logger.DistancesLoggerChannels
}

func (logger *Logger) GetDistancesNumberOfDevices() int {
	return len(*logger.DistancesLoggerChannels)
}

func (logger *Logger) SetDistancesDeviceChannels(channels *DistanceLoggerDeviceChannels) {
	logger.DistancesLoggerChannels = channels
}

func (logger *Logger) addNewDistanceEntry(entry *helpers.DistanceEntry) {
	logger.DistanceEntries = append(logger.DistanceEntries, entry)
}

func (logger *Logger) logDistancesSimulationSummary() {
	sort.SliceStable(logger.DistanceEntries, func(i, j int) bool {
		return logger.DistanceEntries[i].GetTimeStamp() < logger.DistanceEntries[j].GetTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/Distances#%s#%s#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		logger.ConsellationName, logger.TimeStep, logger.TotalSimulationTime)

	log.Default().Println("Writing simulation summary to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := helpers.GetRowsFromDistanceEntries(&logger.DistanceEntries)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

const SIMULATION_EVENT_SENT int = 0
const SIMULATION_EVENT_RECEIVED int = 1
const SIMULATION_EVENT_DROPPED int = 2
const SIMULATION_EVENT_DELIVERED int = 3

type SimulationEvent struct {
	TimeStamp  int
	EventType  int
	FromDevice string
	ToDevice   string
	Packet     *connections.Packet
}

type LoggerDeviceChannels []*LoggerDeviceChannel
type LoggerDeviceChannel chan SimulationEvent

func (logger *Logger) GetNumberOfDevices() int {
	return len(*logger.LoggerDeviceChannels)
}

func (logger *Logger) GetDeviceNames() []string {
	return logger.DeviceNames
}

func (logger *Logger) GetDeviceChannels() *LoggerDeviceChannels {
	return logger.LoggerDeviceChannels
}

func (logger *Logger) GetRemainingUnprocessedPackets() int {
	return logger.RemainingUnprocessedPackets
}

func initChannelCases(selectCases *[]reflect.SelectCase, logger ILogger) {
	channels := *logger.GetDeviceChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (logger *Logger) CloseChannels() {
	for _, channel := range *logger.GetDeviceChannels() {
		close(*channel)
	}
}

func (logger *Logger) ProcessEvent(event SimulationEvent, sourceIndx int) {
	eventType := helpers.EVENT_SENT
	packetId := -1
	switch event.EventType {
	case SIMULATION_EVENT_SENT:
		packetId = event.Packet.PacketId
	case SIMULATION_EVENT_RECEIVED:
		eventType = helpers.EVENT_RECEIVED
		packetId = event.Packet.PacketId
	case SIMULATION_EVENT_DELIVERED:
		eventType = helpers.EVENT_DELIVERED
		packetId = event.Packet.PacketId
		logger.RemainingUnprocessedPackets -= 1
	case SIMULATION_EVENT_DROPPED:
		eventType = helpers.EVENT_DROPPED
		packetId = event.Packet.PacketId
		logger.RemainingUnprocessedPackets -= 1
	}

	newEvent := helpers.SimulationEntry{
		TimeStamp:  event.TimeStamp,
		EventType:  eventType,
		FromDevice: event.FromDevice,
		ToDevice:   event.ToDevice,
		PacketId:   packetId,
	}
	logger.Events = append(logger.Events, &newEvent)
	if newEvent.EventType == helpers.EVENT_DELIVERED || newEvent.EventType == helpers.EVENT_DROPPED {
		println("Remaining Unprocessed Packets: ", logger.GetRemainingUnprocessedPackets())
	}
	//println("Processing Event: ", newEvent.EventType, " from: ", event.FromDevice, " to: ", event.ToDevice,
	//	" at: ", event.TimeStamp, " with packetId: ", packetId)
}

func startLogger(logger ILogger, wg *sync.WaitGroup) {
	println("Remaining Unprocessed Packets: ", logger.GetRemainingUnprocessedPackets())
	for logger.GetRemainingUnprocessedPackets() > 0 {
		selectDevicesCases := make([]reflect.SelectCase, logger.GetNumberOfDevices())
		initChannelCases(&selectDevicesCases, logger)
		index, value, _ := reflect.Select(selectDevicesCases)
		simulationEvent := value.Interface().(SimulationEvent)
		logger.ProcessEvent(simulationEvent, index)
	}
	logger.CloseChannels()
	logger.logSimulationSummary()
	wg.Done()
}

func (logger *Logger) logSimulationSummary() {
	sort.SliceStable(logger.Events, func(i, j int) bool {
		return logger.Events[i].GetTimeStamp() < logger.Events[j].GetTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/SimulationSummary#%s#%s#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		logger.ConsellationName, logger.TimeStep, logger.TotalSimulationTime)

	log.Default().Println("Writing simulation summary to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := helpers.GetRowsFromEvents(&logger.Events)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

func (logger *Logger) Run(wg *sync.WaitGroup) {
	log.Default().Println("Running Logger...")
	go startLogger(logger, wg)
}

func (logger *Logger) SetDeviceChannels(channels *LoggerDeviceChannels, deviceNames []string) {
	logger.LoggerDeviceChannels = channels
	logger.DeviceNames = deviceNames
}
