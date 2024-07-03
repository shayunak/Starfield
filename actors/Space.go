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
	SpaceSatelliteChannels *SpaceSatelliteChannels
	SatelliteNames         []string
	// Distances Mode
	DistancesSpaceSatelliteChannels *DistanceSpaceSatelliteChannels
	DistanceEntries                 helpers.DistanceEntryList
}

type UpdateDistancesMessage struct {
	SatelliteName    string
	SatelliteAnomaly float64
	TimeStamp        int
	Distances        map[string]float64
}

type LinkChannelRequest struct {
	SourceId          string
	DestId            string
	DestTypeSatellite bool
	RecieveChannel    *chan connections.Packet
	SendChannel       *chan connections.Packet
}

type DistanceSpaceSatelliteChannel chan UpdateDistancesMessage
type SpaceSatelliteChannel chan LinkChannelRequest

type DistanceSpaceSatelliteChannels []*DistanceSpaceSatelliteChannel
type SpaceSatelliteChannels []*SpaceSatelliteChannel

type ISpace interface {
	// Distances Mode
	RunDistances(wg *sync.WaitGroup)
	GetDistancesSatelliteChannels() *DistanceSpaceSatelliteChannels
	SetDistancesSatelliteChannels(channels *DistanceSpaceSatelliteChannels)
	GetDistancesNumberOfSatellites() int
	addNewDistanceEntries(distancesMessage UpdateDistancesMessage)
	addNewDistanceEntry(entry *helpers.DistanceEntry)
	logDistancesSimulationSummary()
	// Simulation Mode
	SetSatelliteChannels(channels *SpaceSatelliteChannels, satelliteNames []string)
	GetNumberOfSatellites() int
	GetSatelliteChannels() *SpaceSatelliteChannels
	GetSatelliteNames() []string
	startNewLink(linkRequest LinkChannelRequest, sourceIndex int)
	Run(wg *sync.WaitGroup)
	// General
	GetTotalSimulationTime() int
}

func initDistancesChannelCases(selectCases *[]reflect.SelectCase, space ISpace) {
	channels := *space.GetDistancesSatelliteChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func deleteDistancesSatellite(space ISpace, index int) {
	satellites := *space.GetDistancesSatelliteChannels()
	satellites = append(satellites[:index], satellites[index+1:]...)
	space.SetDistancesSatelliteChannels(&satellites)
}

func startDistancesSpace(space ISpace, wg *sync.WaitGroup) {
	for space.GetDistancesNumberOfSatellites() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, space.GetDistancesNumberOfSatellites())
		initDistancesChannelCases(&selectSatellitesCases, space)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			deleteDistancesSatellite(space, chosen)
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
	satellites := distancesMessage.Distances
	for satelliteId, distance := range satellites {
		space.addNewDistanceEntry(&helpers.DistanceEntry{
			TimeStamp:         distancesMessage.TimeStamp,
			FirstSatelliteId:  distancesMessage.SatelliteName,
			SecondSatelliteId: satelliteId,
			Distance:          distance,
		})
	}
	if space.TimeStamp < distancesMessage.TimeStamp {
		space.TimeStamp = distancesMessage.TimeStamp
	}
}

func (space *Space) GetDistancesSatelliteChannels() *DistanceSpaceSatelliteChannels {
	return space.DistancesSpaceSatelliteChannels
}

func (space *Space) GetDistancesNumberOfSatellites() int {
	return len(*space.DistancesSpaceSatelliteChannels)
}

func (space *Space) SetDistancesSatelliteChannels(channels *DistanceSpaceSatelliteChannels) {
	space.DistancesSpaceSatelliteChannels = channels
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

	rows := helpers.GetRowsFromEvents(&space.DistanceEntries)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

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

func startSpace(space ISpace, wg *sync.WaitGroup) {
	for space.GetNumberOfSatellites() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, space.GetNumberOfSatellites())
		initChannelCases(&selectSatellitesCases, space)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			deleteSatellite(space, chosen)
		}
		newLinkRequest := value.Interface().(LinkChannelRequest)
		space.startNewLink(newLinkRequest, chosen)
	}
	wg.Done()
}

func (space *Space) Run(wg *sync.WaitGroup) {
	log.Default().Println("Running space...")
	go startSpace(space, wg)
}

func (space *Space) GetTotalSimulationTime() int {
	return space.TotalSimulationTime
}

func (space *Space) SetSatelliteChannels(channels *SpaceSatelliteChannels, satelliteNames []string) {
	space.SpaceSatelliteChannels = channels
	space.SatelliteNames = satelliteNames
}

func (space *Space) GetNumberOfSatellites() int {
	return len(*space.SpaceSatelliteChannels)
}

func (space *Space) GetSatelliteNames() []string {
	return space.SatelliteNames
}

func (space *Space) GetSatelliteChannels() *SpaceSatelliteChannels {
	return space.SpaceSatelliteChannels
}
