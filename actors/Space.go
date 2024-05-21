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
)

type Space struct {
	TotalSimulationTime    int // in seconds
	SpaceSatelliteChannels *SpaceSatelliteChannels
	Events                 EventList
	ConsellationName       string
	TimeStep               int
	TimeStamp              int
}

type Event struct {
	TimeStamp         int
	FirstSatelliteId  string
	SecondSatelliteId string
	Distance          float64
}

type UpdateDistancesMessage struct {
	SatelliteName string
	TimeStamp     int
	Distances     map[string]float64
}

type SpaceSatelliteChannel chan UpdateDistancesMessage

type SpaceSatelliteChannels []*SpaceSatelliteChannel

type EventList []IEvent

type ISpace interface {
	Run(wg *sync.WaitGroup)
	SetSatelliteChannels(channels *SpaceSatelliteChannels)
	GetNumberOfSatellites() int
	GetSatelliteChannels() *SpaceSatelliteChannels
	GetTotalSimulationTime() int
	addNewEvents(distancesMessage UpdateDistancesMessage)
	addNewEvent(event *Event)
	logSimulationSummary()
}

type IEvent interface {
	getHeaders() []string
	toSlice() []string
	getTimeStamp() int
}

func (event *Event) getTimeStamp() int {
	return event.TimeStamp
}

func (event *Event) getHeaders() []string {
	return []string{"TimeStamp", "FirstSatelliteId", "SecondSatelliteId", "Distance"}
}

func (event *Event) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", event.TimeStamp),
		event.FirstSatelliteId,
		event.SecondSatelliteId,
		fmt.Sprintf("%f", event.Distance),
	}
}

func initChannelCases(selectCases *[]reflect.SelectCase, space ISpace) {
	channels := *space.GetSatelliteChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func deleteSatellite(space ISpace, index int) {
	satellites := *space.GetSatelliteChannels()
	satellites = append(satellites[:index], satellites[index+1:]...)
	space.SetSatelliteChannels(&satellites)
}

func (space *Space) addNewEvents(distancesMessage UpdateDistancesMessage) {
	satellites := distancesMessage.Distances
	for satelliteId, distance := range satellites {
		space.addNewEvent(&Event{
			TimeStamp:         distancesMessage.TimeStamp,
			FirstSatelliteId:  distancesMessage.SatelliteName,
			SecondSatelliteId: satelliteId,
			Distance:          distance,
		})
	}
	if space.TimeStamp < distancesMessage.TimeStamp {
		space.TimeStamp = distancesMessage.TimeStamp
		log.Default().Printf("Now moving to Timestamp: %d\n", space.TimeStamp)
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
		distanceUpdateMessage := value.Interface().(UpdateDistancesMessage)
		space.addNewEvents(distanceUpdateMessage)
	}
	space.logSimulationSummary()
	wg.Done()
}

func (space *Space) Run(wg *sync.WaitGroup) {
	log.Default().Println("Running space...")
	go startSpace(space, wg)
}

func (space *Space) GetTotalSimulationTime() int {
	return space.TotalSimulationTime
}

func (space *Space) SetSatelliteChannels(channels *SpaceSatelliteChannels) {
	space.SpaceSatelliteChannels = channels
}

func (space *Space) GetNumberOfSatellites() int {
	return len(*space.SpaceSatelliteChannels)
}

func (space *Space) addNewEvent(event *Event) {
	space.Events = append(space.Events, event)
}

func (space *Space) GetSatelliteChannels() *SpaceSatelliteChannels {
	return space.SpaceSatelliteChannels
}

func getRowsFromEvents(events *EventList) [][]string {
	var rows [][]string
	for i, event := range *events {
		if i == 0 {
			rows = append(rows, event.getHeaders())
		}
		rows = append(rows, event.toSlice())
	}
	return rows
}

func (space *Space) logSimulationSummary() {
	sort.SliceStable(space.Events, func(i, j int) bool {
		return space.Events[i].getTimeStamp() < space.Events[j].getTimeStamp()
	}) // Sorting events by timestamp

	if _, err := os.Stat("./generated"); os.IsNotExist(err) {
		err := os.Mkdir("./generated", 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fileName := fmt.Sprintf("./generated/%s#%s#%dms#%ds.csv", time.Now().Format("2006_01_02,15_04_05"),
		space.ConsellationName, space.TimeStep, space.TotalSimulationTime/1000)

	log.Default().Println("Writing simulation summary to ", fileName)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	rows := getRowsFromEvents(&space.Events)
	csvWriter := csv.NewWriter(outputFile)

	if err := csvWriter.WriteAll(rows); err != nil {
		log.Fatal(err)
	}

	outputFile.Close()
}
