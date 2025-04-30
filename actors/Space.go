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

type Space struct {
	ConsellationName    string
	TimeStep            int
	TimeStamp           int
	TotalSimulationTime int // in seconds
	// Simulation Mode
	SpaceSatelliteChannels      *SpaceSatelliteChannels
	RemainingUnprocessedPackets int
	SatelliteNames              []string
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
	SetSatelliteChannels(channels *SpaceSatelliteChannels, satelliteNames []string)
	GetNumberOfSatellites() int
	GetSatelliteChannels() *SpaceSatelliteChannels
	GetSatelliteNames() []string
	GetRemainingUnprocessedPackets() int
	ProcessEvent(event SimulationEvent)
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
	SourceId          string
	DestId            string
	DestTypeSatellite bool
	RecieveChannel    *chan connections.Packet
	SendChannel       *chan connections.Packet
}

const SIMULATION_EVENT_SENT int = 0
const SIMULATION_EVENT_RECEIVED int = 1
const SIMULATION_EVENT_DROPPED int = 2
const SIMULATION_EVENT_CONNECTION_LOST int = 3
const SIMULATION_EVENT_CONNECTION_ESTABLISHED int = 4

type SimulationEvent struct {
	TimeStamp     int
	EventType     int
	FromSatellite string
	ToSatellite   string
	Packet        *connections.Packet
}

type SpaceSatelliteChannels []*SpaceSatelliteChannel
type SpaceSatelliteChannel chan SimulationEvent

func (space *Space) GetNumberOfSatellites() int {
	return len(*space.SpaceSatelliteChannels)
}

func (space *Space) GetSatelliteNames() []string {
	return space.SatelliteNames
}

func (space *Space) GetSatelliteChannels() *SpaceSatelliteChannels {
	return space.SpaceSatelliteChannels
}

func (space *Space) GetRemainingUnprocessedPackets() int {
	return space.RemainingUnprocessedPackets
}

func initChannelCases(selectCases *[]reflect.SelectCase, space ISpace) {
	channels := *space.GetSatelliteChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func deleteSatellite(space ISpace, index int) {
	satellites := *space.GetSatelliteChannels()
	names := space.GetSatelliteNames()
	satellites = append(satellites[:index], satellites[index+1:]...)
	names = append(names[:index], names[index+1:]...)
	space.SetSatelliteChannels(&satellites, names)
}

func (space *Space) ProcessEvent(event SimulationEvent) {
	eventType := helpers.EVENT_SENT
	packetId := -1
	switch event.EventType {
	case SIMULATION_EVENT_SENT:
		packetId = event.Packet.PacketId
	case SIMULATION_EVENT_RECEIVED:
		eventType = helpers.EVENT_RECEIVED
		packetId = event.Packet.PacketId
		if event.Packet.Destination == event.ToSatellite {
			space.RemainingUnprocessedPackets -= 1
		}
	case SIMULATION_EVENT_DROPPED:
		eventType = helpers.EVENT_DROPPED
		packetId = event.Packet.PacketId
		space.RemainingUnprocessedPackets -= 1
	case SIMULATION_EVENT_CONNECTION_LOST:
		eventType = helpers.EVENT_CONNECTION_LOST
	case SIMULATION_EVENT_CONNECTION_ESTABLISHED:
		eventType = helpers.EVENT_CONNECTION_ESTABLISHED
	}

	newEvent := helpers.SimulationEntry{
		TimeStamp:  event.TimeStamp,
		EventType:  eventType,
		FromDevice: event.FromSatellite,
		ToDevice:   event.ToSatellite,
		PacketId:   packetId,
	}
	space.Events = append(space.Events, &newEvent)
}

/*
Link request is not working right now, since the channel cannot handle two types of strctural messages.

func (space *Space) startNewLink(linkRequest LinkChannelRequest, sourceIndex int) {
	satelliteNames := space.GetSatelliteNames()
	satellites := *space.GetSatelliteChannels()
	destIndex := slices.IndexFunc(satelliteNames, func(name string) bool { return name == linkRequest.DestId })
	if destIndex == -1 {
		log.Default().Println("Unknown destination: ", linkRequest.DestId)
		*satellites[sourceIndex] <- linkRequest
	} else {
		*satellites[destIndex] <- linkRequest
	}
}
*/

func startSpace(space ISpace, wg *sync.WaitGroup) {
	for space.GetRemainingUnprocessedPackets() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, space.GetNumberOfSatellites())
		initChannelCases(&selectSatellitesCases, space)
		_, value, _ := reflect.Select(selectSatellitesCases)
		simulationEvent := value.Interface().(SimulationEvent)
		space.ProcessEvent(simulationEvent)
	}
	wg.Done()
}

func (space *Space) logSimulationSummary() {
	sort.SliceStable(space.Events, func(i, j int) bool {
		return space.Events[i].GetTimeStamp() < space.DistanceEntries[j].GetTimeStamp()
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

	rows := helpers.GetRowsFromDistanceEntries(&space.DistanceEntries)
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

func (space *Space) SetSatelliteChannels(channels *SpaceSatelliteChannels, satelliteNames []string) {
	space.SpaceSatelliteChannels = channels
	space.SatelliteNames = satelliteNames
}
