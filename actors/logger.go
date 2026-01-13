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

	"SatSimGo/connections"

	"SatSimGo/helpers"
)

type Logger struct {
	ConsellationName           string
	TimeStep                   int
	TimeStamp                  float64 // in milliseconds
	TotalSimulationTime        float64 // in milliseconds
	NumberOfOrbits             int
	NumberOfSatellitesPerOrbit int
	// Simulation Mode
	CoordinatorChannel          *chan float64
	LoggerDeviceChannels        *LoggerDeviceChannels
	RemainingUnprocessedPackets int
	DeviceNames                 []string
	Events                      helpers.SimulationEntryList
	// Distances Mode
	DistancesLoggerChannels *DistanceLoggerDeviceChannels
	DistanceEntries         helpers.DistanceEntryList
	// Positions Mode
	SphericalPositionsLoggerChannels *SphericalPositionLoggerDeviceChannels
	CartesianPositionsLoggerChannels *CartesianPositionLoggerDeviceChannels
	SphericalPositionEntries         helpers.PositionEntryList
	CartesianPositionEntries         helpers.PositionEntryList
}

type DistanceLoggerDeviceChannel chan UpdateDistancesMessage
type DistanceLoggerDeviceChannels []*DistanceLoggerDeviceChannel

type SphericalPositionLoggerDeviceChannel chan UpdateSphericalPositionMessage
type SphericalPositionLoggerDeviceChannels []*SphericalPositionLoggerDeviceChannel

type CartesianPositionLoggerDeviceChannel chan UpdateCartesianPositionMessage
type CartesianPositionLoggerDeviceChannels []*CartesianPositionLoggerDeviceChannel

type ILogger interface {
	// Positions Mode
	RunSphericalPositions(wg *sync.WaitGroup)
	RunCartesianPositions(wg *sync.WaitGroup)
	SetSphericalPositionsDeviceChannels(channels *SphericalPositionLoggerDeviceChannels)
	SetCartesianPositionsDeviceChannels(channels *CartesianPositionLoggerDeviceChannels)
	GetSphericalPositionsNumberOfDevices() int
	GetCartesianPositionsNumberOfDevices() int
	InitSphericalPositionsChannelCases(selectCases *[]reflect.SelectCase)
	InitCartesianPositionsChannelCases(selectCases *[]reflect.SelectCase)
	DeleteSphericalPositionsDevice(index int)
	DeleteCartesianPositionsDevice(index int)
	addNewSphericalPositionEntry(positionsMessage UpdateSphericalPositionMessage)
	addNewCartesianPositionEntry(positionsMessage UpdateCartesianPositionMessage)
	logSphericalPositionsSimulationSummary()
	logCartesianPositionsSimulationSummary()
	// Distances Mode
	RunDistances(wg *sync.WaitGroup)
	SetDistancesDeviceChannels(channels *DistanceLoggerDeviceChannels)
	GetDistancesNumberOfDevices() int
	InitDistancesChannelCases(selectCases *[]reflect.SelectCase)
	DeleteDistancesDevice(index int)
	addNewDistanceEntries(distancesMessage UpdateDistancesMessage)
	addNewDistanceEntry(entry *helpers.DistanceEntry)
	logDistancesSimulationSummary()
	// Simulation Mode
	UpdateTimeStamp(newTimeStamp float64)
	SetDeviceChannels(channels *LoggerDeviceChannels, deviceNames []string)
	GetNumberOfDevices() int
	GetDeviceChannels() *LoggerDeviceChannels
	GetDeviceNames() []string
	GetRemainingUnprocessedPackets() int
	ProcessEvent(event SimulationEvent, sourceIndex int)
	InitChannelCases(selectCases *[]reflect.SelectCase)
	logSimulationSummary()
	Run(wg *sync.WaitGroup)
	// General
	GetTotalSimulationTime() float64
	GetTimeStamp() float64
}

func (logger *Logger) GetTotalSimulationTime() float64 {
	return logger.TotalSimulationTime
}

func (logger *Logger) GetTimeStamp() float64 {
	return logger.TimeStamp
}

//////////////////////////////////// ****** Positions Mode ****** //////////////////////////////////////////////////

type UpdateSphericalPositionMessage struct {
	DeviceName string
	TimeStamp  float64 // in milliseconds
	Spherical  helpers.SphericalCoordinates
}

type UpdateCartesianPositionMessage struct {
	DeviceName string
	TimeStamp  float64 // in milliseconds
	Cartesian  helpers.CartesianCoordinates
}

func (logger *Logger) InitSphericalPositionsChannelCases(selectCases *[]reflect.SelectCase) {
	channels := *logger.SphericalPositionsLoggerChannels
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (logger *Logger) InitCartesianPositionsChannelCases(selectCases *[]reflect.SelectCase) {
	channels := *logger.CartesianPositionsLoggerChannels
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (logger *Logger) DeleteSphericalPositionsDevice(index int) {
	devices := *logger.SphericalPositionsLoggerChannels
	devices = append(devices[:index], devices[index+1:]...)
	logger.SetSphericalPositionsDeviceChannels(&devices)
}

func (logger *Logger) DeleteCartesianPositionsDevice(index int) {
	devices := *logger.CartesianPositionsLoggerChannels
	devices = append(devices[:index], devices[index+1:]...)
	logger.SetCartesianPositionsDeviceChannels(&devices)
}

func startSphericalPositionsLogger(logger ILogger, wg *sync.WaitGroup) {
	for logger.GetSphericalPositionsNumberOfDevices() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, logger.GetSphericalPositionsNumberOfDevices())
		logger.InitSphericalPositionsChannelCases(&selectSatellitesCases)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			logger.DeleteSphericalPositionsDevice(chosen)
			continue
		}
		positionUpdateMessage := value.Interface().(UpdateSphericalPositionMessage)
		logger.addNewSphericalPositionEntry(positionUpdateMessage)
	}
	logger.logSphericalPositionsSimulationSummary()
	wg.Done()
}

func startCartesianPositionsLogger(logger ILogger, wg *sync.WaitGroup) {
	for logger.GetCartesianPositionsNumberOfDevices() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, logger.GetCartesianPositionsNumberOfDevices())
		logger.InitCartesianPositionsChannelCases(&selectSatellitesCases)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			logger.DeleteCartesianPositionsDevice(chosen)
			continue
		}
		positionUpdateMessage := value.Interface().(UpdateCartesianPositionMessage)
		logger.addNewCartesianPositionEntry(positionUpdateMessage)
	}
	logger.logCartesianPositionsSimulationSummary()
	wg.Done()
}

func (logger *Logger) RunSphericalPositions(wg *sync.WaitGroup) {
	log.Default().Println("Running Spherical Position Analyzer...")
	go startSphericalPositionsLogger(logger, wg)
}

func (logger *Logger) RunCartesianPositions(wg *sync.WaitGroup) {
	log.Default().Println("Running Cartesian Position Analyzer...")
	go startCartesianPositionsLogger(logger, wg)
}

func (logger *Logger) addNewSphericalPositionEntry(positionsMessage UpdateSphericalPositionMessage) {
	logger.SphericalPositionEntries = append(logger.SphericalPositionEntries, &helpers.SphericalPositionEntry{
		TimeStamp: int(positionsMessage.TimeStamp),
		Id:        positionsMessage.DeviceName,
		Latitude:  positionsMessage.Spherical.Latitude,
		Longitude: positionsMessage.Spherical.Longitude,
		Radius:    positionsMessage.Spherical.Radius,
	})

	if logger.TimeStamp < positionsMessage.TimeStamp {
		logger.TimeStamp = positionsMessage.TimeStamp
	}
}

func (logger *Logger) addNewCartesianPositionEntry(positionsMessage UpdateCartesianPositionMessage) {
	logger.CartesianPositionEntries = append(logger.CartesianPositionEntries, &helpers.CartesianPositionEntry{
		TimeStamp: int(positionsMessage.TimeStamp),
		Id:        positionsMessage.DeviceName,
		X:         positionsMessage.Cartesian.X,
		Y:         positionsMessage.Cartesian.Y,
		Z:         positionsMessage.Cartesian.Z,
	})

	if logger.TimeStamp < positionsMessage.TimeStamp {
		logger.TimeStamp = positionsMessage.TimeStamp
	}
}

func (logger *Logger) GetSphericalPositionsNumberOfDevices() int {
	return len(*logger.SphericalPositionsLoggerChannels)
}

func (logger *Logger) GetCartesianPositionsNumberOfDevices() int {
	return len(*logger.CartesianPositionsLoggerChannels)
}

func (logger *Logger) SetSphericalPositionsDeviceChannels(channels *SphericalPositionLoggerDeviceChannels) {
	logger.SphericalPositionsLoggerChannels = channels
}

func (logger *Logger) SetCartesianPositionsDeviceChannels(channels *CartesianPositionLoggerDeviceChannels) {
	logger.CartesianPositionsLoggerChannels = channels
}

func (logger *Logger) logSphericalPositionsSimulationSummary() {
	sort.SliceStable(logger.SphericalPositionEntries, func(i, j int) bool {
		return logger.SphericalPositionEntries[i].GetTimeStamp() < logger.SphericalPositionEntries[j].GetTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/SphericalPositions#%s#%s(%d,%d)#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		logger.ConsellationName, logger.NumberOfOrbits, logger.NumberOfSatellitesPerOrbit, logger.TimeStep, int(logger.TotalSimulationTime/1000.0))

	log.Default().Println("Writing positions to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := helpers.GetRowsFromPositionEntries(&logger.SphericalPositionEntries)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

func (logger *Logger) logCartesianPositionsSimulationSummary() {
	sort.SliceStable(logger.CartesianPositionEntries, func(i, j int) bool {
		return logger.CartesianPositionEntries[i].GetTimeStamp() < logger.CartesianPositionEntries[j].GetTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/CartesianPositions#%s#%s(%d,%d)#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		logger.ConsellationName, logger.NumberOfOrbits, logger.NumberOfSatellitesPerOrbit, logger.TimeStep, int(logger.TotalSimulationTime/1000.0))

	log.Default().Println("Writing positions to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := helpers.GetRowsFromPositionEntries(&logger.CartesianPositionEntries)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

type UpdateDistancesMessage struct {
	DeviceName string
	TimeStamp  float64 // in milliseconds
	Distances  map[string]float64
}

func (logger *Logger) InitDistancesChannelCases(selectCases *[]reflect.SelectCase) {
	channels := *logger.DistancesLoggerChannels
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (logger *Logger) DeleteDistancesDevice(index int) {
	devices := *logger.DistancesLoggerChannels
	devices = append(devices[:index], devices[index+1:]...)
	logger.SetDistancesDeviceChannels(&devices)
}

func startDistancesLogger(logger ILogger, wg *sync.WaitGroup) {
	for logger.GetDistancesNumberOfDevices() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, logger.GetDistancesNumberOfDevices())
		logger.InitDistancesChannelCases(&selectSatellitesCases)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			logger.DeleteDistancesDevice(chosen)
			continue
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
			TimeStamp:  int(distancesMessage.TimeStamp),
			FromDevice: distancesMessage.DeviceName,
			ToDevice:   deviceId,
			Distance:   int(distance),
		})
	}
	if logger.TimeStamp < distancesMessage.TimeStamp {
		logger.TimeStamp = distancesMessage.TimeStamp
	}
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

	fileName := fmt.Sprintf("./generated/Distances#%s#%s(%d,%d)#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		logger.ConsellationName, logger.NumberOfOrbits, logger.NumberOfSatellitesPerOrbit, logger.TimeStep, int(logger.TotalSimulationTime/1000.0))

	log.Default().Println("Writing distances to ", fileName)
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

func (logger *Logger) UpdateTimeStamp(newTimeStamp float64) {
	logger.TimeStamp = newTimeStamp
}

func (logger *Logger) GetDeviceChannels() *LoggerDeviceChannels {
	return logger.LoggerDeviceChannels
}

func (logger *Logger) GetRemainingUnprocessedPackets() int {
	return logger.RemainingUnprocessedPackets
}

func (logger *Logger) InitChannelCases(selectCases *[]reflect.SelectCase) {
	channels := *logger.GetDeviceChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
	(*selectCases)[logger.GetNumberOfDevices()] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*logger.CoordinatorChannel)}
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
}

func startLogger(logger ILogger, wg *sync.WaitGroup) {
	for logger.GetRemainingUnprocessedPackets() > 0 {
		selectDevicesCases := make([]reflect.SelectCase, logger.GetNumberOfDevices()+1)
		logger.InitChannelCases(&selectDevicesCases)
		index, value, _ := reflect.Select(selectDevicesCases)
		if index == logger.GetNumberOfDevices() { // Coordinator channel
			timeStamp := value.Interface().(float64)
			logger.UpdateTimeStamp(timeStamp)
			log.Default().Println("Time Progress: ", timeStamp, " ms, Remaining Unprocessed Packets: ", logger.GetRemainingUnprocessedPackets())
		} else {
			simulationEvent := value.Interface().(SimulationEvent)
			logger.ProcessEvent(simulationEvent, index)
		}
	}
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

	fileName := fmt.Sprintf("./generated/SimulationSummary#%s#%s(%d,%d)#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		logger.ConsellationName, logger.NumberOfOrbits, logger.NumberOfSatellitesPerOrbit, logger.TimeStep, int(logger.TotalSimulationTime/1000.0))

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
