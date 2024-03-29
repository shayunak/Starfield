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
	TotalSimulationTime    int // in milliseconds
	SpaceSatelliteChannels *SpaceSatelliteChannels
	Events                 EventList
	ConsellationName       string
	TimeStep               int
}

type Event struct {
	TimeStamp   int
	satelliteId string
	orbitId     string
	X           float64
	Y           float64
	Z           float64
	Anomaly     float64
}

type UpdatePoisitionMessage struct {
	SatelliteId string
	OrbitId     string
	Position    CartesianCoordinates
	Anomaly     float64
	TimeStamp   int
}

type SpaceSatelliteChannel chan UpdatePoisitionMessage

type SpaceSatelliteChannels []*SpaceSatelliteChannel

type EventList []IEvent

type ISpace interface {
	Run(wg *sync.WaitGroup)
	SetSatelliteChannels(channels *SpaceSatelliteChannels)
	GetNumberOfSatellites() int
	GetSatelliteChannels() *SpaceSatelliteChannels
	GetTotalSimulationTime() int
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
	return []string{"TimeStamp", "satelliteId", "orbitId", "X", "Y", "Z", "Anomaly"}
}

func (event *Event) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", event.TimeStamp),
		event.satelliteId,
		event.orbitId,
		fmt.Sprintf("%f", event.X),
		fmt.Sprintf("%f", event.Y),
		fmt.Sprintf("%f", event.Z),
		fmt.Sprintf("%f", event.Anomaly),
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
	close(*satellites[index])
	satellites = append(satellites[:index], satellites[index+1:]...)
	space.SetSatelliteChannels(&satellites)
}

func startSpace(space ISpace, wg *sync.WaitGroup) {
	for space.GetNumberOfSatellites() > 0 {
		selectSatellitesCases := make([]reflect.SelectCase, space.GetNumberOfSatellites())
		initChannelCases(&selectSatellitesCases, space)
		chosen, value, ok := reflect.Select(selectSatellitesCases)
		if !ok {
			log.Default().Printf("Chosen channel: %d, unexpectedly closed!\n", chosen)
		}
		positionUpdateMessage := value.Interface().(UpdatePoisitionMessage)
		if positionUpdateMessage.TimeStamp > space.GetTotalSimulationTime() {
			log.Default().Println("Simulation time exceeded for satellite ", positionUpdateMessage.SatelliteId, "!")
			deleteSatellite(space, chosen)
		} else {
			satellites := *space.GetSatelliteChannels()
			*satellites[chosen] <- positionUpdateMessage
			space.addNewEvent(&Event{
				TimeStamp:   positionUpdateMessage.TimeStamp,
				satelliteId: positionUpdateMessage.SatelliteId,
				orbitId:     positionUpdateMessage.OrbitId,
				X:           positionUpdateMessage.Position.X,
				Y:           positionUpdateMessage.Position.Y,
				Z:           positionUpdateMessage.Position.Z,
				Anomaly:     positionUpdateMessage.Anomaly,
			})
		}
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
