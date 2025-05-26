package actors

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"reflect"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
)

type Space struct {
	ConsellationName    string
	TimeStep            int
	TimeStamp           int
	TotalSimulationTime int // in seconds
	// Simulation Mode
	SpaceDeviceChannels         *SpaceDeviceChannels
	RemainingUnprocessedPackets int
	InterfaceBufferSize         int
	DeviceNames                 []string
	Events                      helpers.SimulationEntryList
	// Distances Mode
	DistancesSpaceChannels *DistanceSpaceDeviceChannels
	DistanceEntries        helpers.DistanceEntryList
}

type DistanceSpaceDeviceChannel chan UpdateDistancesMessage
type DistanceSpaceDeviceChannels []*DistanceSpaceDeviceChannel

type ISpace interface {
	// Distances Mode
	RunDistances(wg *sync.WaitGroup)
	GetDistancesDeviceChannels() *DistanceSpaceDeviceChannels
	SetDistancesDeviceChannels(channels *DistanceSpaceDeviceChannels)
	GetDistancesNumberOfDevices() int
	addNewDistanceEntries(distancesMessage UpdateDistancesMessage)
	addNewDistanceEntry(entry *helpers.DistanceEntry)
	logDistancesSimulationSummary()
	// Simulation Mode
	SetDeviceChannels(channels *SpaceDeviceChannels, deviceNames []string)
	GetNumberOfDevices() int
	GetDeviceChannels() *SpaceDeviceChannels
	GetDeviceNames() []string
	GetRemainingUnprocessedPackets() int
	ProcessEvent(event SimulationEvent, sourceIndex int)
	CloseChannels()
	startNewLink(event SimulationEvent, sourceIndex int) bool
	logSimulationSummary()
	//startNewLink(linkRequest LinkChannelRequest, sourceIndex int)
	Run(wg *sync.WaitGroup)
	// General
	GetTotalSimulationTime() int
}

func (space *Space) GetTotalSimulationTime() int {
	return space.TotalSimulationTime
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

type UpdateDistancesMessage struct {
	DeviceName string
	TimeStamp  int
	Distances  map[string]float64
}

func initDistancesChannelCases(selectCases *[]reflect.SelectCase, space ISpace) {
	channels := *space.GetDistancesDeviceChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func deleteDistancesDevice(space ISpace, index int) {
	devices := *space.GetDistancesDeviceChannels()
	devices = append(devices[:index], devices[index+1:]...)
	space.SetDistancesDeviceChannels(&devices)
}

func startDistancesSpace(space ISpace, wg *sync.WaitGroup) {
	for space.GetDistancesNumberOfDevices() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, space.GetDistancesNumberOfDevices())
		initDistancesChannelCases(&selectSatellitesCases, space)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			deleteDistancesDevice(space, chosen)
		}
		distanceUpdateMessage := value.Interface().(UpdateDistancesMessage)
		space.addNewDistanceEntries(distanceUpdateMessage)
	}
	space.logDistancesSimulationSummary()
	wg.Done()
}

func (space *Space) RunDistances(wg *sync.WaitGroup) {
	log.Default().Println("Running Distance Analyzer...")
	go startDistancesSpace(space, wg)
}

func (space *Space) addNewDistanceEntries(distancesMessage UpdateDistancesMessage) {
	devices := distancesMessage.Distances
	for deviceId, distance := range devices {
		space.addNewDistanceEntry(&helpers.DistanceEntry{
			TimeStamp:  distancesMessage.TimeStamp,
			FromDevice: distancesMessage.DeviceName,
			ToDevice:   deviceId,
			Distance:   int(distance),
		})
	}
	if space.TimeStamp < distancesMessage.TimeStamp {
		space.TimeStamp = distancesMessage.TimeStamp
	}
}

func (space *Space) GetDistancesDeviceChannels() *DistanceSpaceDeviceChannels {
	return space.DistancesSpaceChannels
}

func (space *Space) GetDistancesNumberOfDevices() int {
	return len(*space.DistancesSpaceChannels)
}

func (space *Space) SetDistancesDeviceChannels(channels *DistanceSpaceDeviceChannels) {
	space.DistancesSpaceChannels = channels
}

func (space *Space) addNewDistanceEntry(entry *helpers.DistanceEntry) {
	space.DistanceEntries = append(space.DistanceEntries, entry)
}

func (space *Space) logDistancesSimulationSummary() {
	sort.SliceStable(space.DistanceEntries, func(i, j int) bool {
		return space.DistanceEntries[i].GetTimeStamp() < space.DistanceEntries[j].GetTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/Distances#%s#%s#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		space.ConsellationName, space.TimeStep, space.TotalSimulationTime)

	log.Default().Println("Writing simulation summary to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := helpers.GetRowsFromDistanceEntries(&space.DistanceEntries)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

type LinkChannelRequest struct {
	SendChannel    *chan connections.Packet
	RecieveChannel *chan connections.Packet
}

const SIMULATION_EVENT_SENT int = 0
const SIMULATION_EVENT_RECEIVED int = 1
const SIMULATION_EVENT_DROPPED int = 2
const SIMULATION_EVENT_DELIVERED int = 3
const SIMULATION_EVENT_CONNECTION_ESTABLISHED int = 4

type SimulationEvent struct {
	TimeStamp  int
	EventType  int
	FromDevice string
	ToDevice   string
	Packet     *connections.Packet
	LinkReq    *LinkChannelRequest
}

type SpaceDeviceChannels []*SpaceDeviceChannel
type SpaceDeviceChannel chan SimulationEvent

func (space *Space) GetNumberOfDevices() int {
	return len(*space.SpaceDeviceChannels)
}

func (space *Space) GetDeviceNames() []string {
	return space.DeviceNames
}

func (space *Space) GetDeviceChannels() *SpaceDeviceChannels {
	return space.SpaceDeviceChannels
}

func (space *Space) GetRemainingUnprocessedPackets() int {
	return space.RemainingUnprocessedPackets
}

func initChannelCases(selectCases *[]reflect.SelectCase, space ISpace) {
	channels := *space.GetDeviceChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (space *Space) CloseChannels() {
	for _, channel := range *space.GetDeviceChannels() {
		close(*channel)
	}
}

func (space *Space) ProcessEvent(event SimulationEvent, sourceIndx int) {
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
		space.RemainingUnprocessedPackets -= 1
	case SIMULATION_EVENT_DROPPED:
		eventType = helpers.EVENT_DROPPED
		packetId = event.Packet.PacketId
		space.RemainingUnprocessedPackets -= 1
	case SIMULATION_EVENT_CONNECTION_ESTABLISHED:
		eventType = helpers.EVENT_CONNECTION_ESTABLISHED
		success := space.startNewLink(event, sourceIndx)
		if !success {
			return
		}
	}

	newEvent := helpers.SimulationEntry{
		TimeStamp:  event.TimeStamp,
		EventType:  eventType,
		FromDevice: event.FromDevice,
		ToDevice:   event.ToDevice,
		PacketId:   packetId,
	}
	space.Events = append(space.Events, &newEvent)
}

func (space *Space) startNewLink(event SimulationEvent, sourceIndex int) bool {
	deviceNames := space.GetDeviceNames()
	channels := *space.GetDeviceChannels()
	destIndex := slices.IndexFunc(deviceNames, func(name string) bool { return name == event.ToDevice })
	if destIndex == -1 {
		log.Default().Println("Unknown destination: ", event.ToDevice)
		return false
	}
	sendPacketChannel := make(chan connections.Packet, space.InterfaceBufferSize)
	receivePacketChannel := make(chan connections.Packet, space.InterfaceBufferSize)

	*channels[destIndex] <- SimulationEvent{
		TimeStamp:  event.TimeStamp,
		EventType:  SIMULATION_EVENT_CONNECTION_ESTABLISHED,
		ToDevice:   event.FromDevice,
		FromDevice: event.ToDevice,
		LinkReq: &LinkChannelRequest{
			SendChannel:    &receivePacketChannel,
			RecieveChannel: &sendPacketChannel,
		},
	}
	*channels[sourceIndex] <- SimulationEvent{
		TimeStamp:  event.TimeStamp,
		EventType:  SIMULATION_EVENT_CONNECTION_ESTABLISHED,
		ToDevice:   event.ToDevice,
		FromDevice: event.FromDevice,
		LinkReq: &LinkChannelRequest{
			SendChannel:    &sendPacketChannel,
			RecieveChannel: &receivePacketChannel,
		},
	}
	return true
}

func startSpace(space ISpace, wg *sync.WaitGroup) {
	for space.GetRemainingUnprocessedPackets() > 0 {
		selectDevicesCases := make([]reflect.SelectCase, space.GetNumberOfDevices())
		initChannelCases(&selectDevicesCases, space)
		index, value, _ := reflect.Select(selectDevicesCases)
		simulationEvent := value.Interface().(SimulationEvent)
		space.ProcessEvent(simulationEvent, index)
	}
	space.CloseChannels()
	space.logSimulationSummary()
	wg.Done()
}

func (space *Space) logSimulationSummary() {
	sort.SliceStable(space.Events, func(i, j int) bool {
		return space.Events[i].GetTimeStamp() < space.Events[j].GetTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/SimulationSummary#%s#%s#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		space.ConsellationName, space.TimeStep, space.TotalSimulationTime)

	log.Default().Println("Writing simulation summary to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := helpers.GetRowsFromEvents(&space.Events)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

func (space *Space) Run(wg *sync.WaitGroup) {
	log.Default().Println("Running space...")
	go startSpace(space, wg)
}

func (space *Space) SetDeviceChannels(channels *SpaceDeviceChannels, deviceNames []string) {
	space.SpaceDeviceChannels = channels
	space.DeviceNames = deviceNames
}
