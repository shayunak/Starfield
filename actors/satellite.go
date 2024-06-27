package actors

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
)

type TrafficEntry struct {
	Destination string
	TimeStamp   int // in milliseconds
	Length      int // in bits
}

type ForwardingEntry map[string]string

type Satellite struct {
	Name string
	Id   int
	// Position            helpers.CartesianCoordinates (Unnecessary for satellite distances calculations)
	TotalSimulationTime  int // in milliseconds
	AnomalyElements      helpers.AnomalyElements
	Orbit                helpers.IOrbit
	DistanceSpaceChannel *DistanceSpaceSatelliteChannel
	SpaceChannel         *SpaceSatelliteChannel
	Dt                   int     // in milliseconds
	TimeStamp            int     // in milliseconds
	OrbitalAnomaly       float64 // in radians
	ForwardingFile       string
	ForwardingTable      map[int]ForwardingEntry
	AnomalyCalculations  helpers.IAnomalyCalculation
	ISLInterfaces        []connections.INetworkInterface
}

type ISatellite interface {
	Run()
	RunDistances()
	SetForwardingFile(folder string)
	GetSpaceChannel() *SpaceSatelliteChannel
	SetSpaceChannel(channel *SpaceSatelliteChannel)
	GetDistanceSpaceChannel() *DistanceSpaceSatelliteChannel
	SetDistanceSpaceChannel(channel *DistanceSpaceSatelliteChannel)
	GetName() string
	GenerateTraffic(traffic []TrafficEntry)
	getTimeStamp() int
	getTotalSimulationTime() int
	updatePosition()
	updateSpaceOnDistances()
	nextTimeStep()
	loadForwardingTableInMemory()
}

func (satellite *Satellite) SetForwardingFile(folder string) {
	satellite.ForwardingFile = fmt.Sprintf("./forwarding_table/%s/%s.csv", folder, satellite.Name)
}

func (satellite *Satellite) GetName() string {
	return satellite.Name
}

func (satellite *Satellite) getTimeStamp() int {
	return satellite.TimeStamp
}

func (satellite *Satellite) getTotalSimulationTime() int {
	return satellite.TotalSimulationTime
}

func (satellite *Satellite) GenerateTraffic(traffic []TrafficEntry) {

}

func (satellite *Satellite) RunDistances() {
	log.Default().Println("Running satellite (Distance Mode): ", satellite.Id)
	go startSatelliteDistances(satellite)
}

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Id)
	go startSatellite(satellite)
}

func (satellite *Satellite) loadForwardingTableInMemory() {
	satellite.ForwardingTable = make(map[int]ForwardingEntry)

	file, err := os.Open(satellite.ForwardingFile)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)

	// ignore the header
	_, _ = reader.Read()
	// read the data
	records, _ := reader.ReadAll()

	for _, record := range records {
		timeStamp, _ := strconv.Atoi(record[0])
		if satellite.ForwardingTable[timeStamp] == nil {
			satellite.ForwardingTable[timeStamp] = make(ForwardingEntry)
		}
		satellite.ForwardingTable[timeStamp][record[1]] = record[2]
	}
}

func (satellite *Satellite) GetSpaceChannel() *SpaceSatelliteChannel {
	return satellite.SpaceChannel
}

func (satellite *Satellite) SetSpaceChannel(channel *SpaceSatelliteChannel) {
	satellite.SpaceChannel = channel
}

func (satellite *Satellite) GetDistanceSpaceChannel() *DistanceSpaceSatelliteChannel {
	return satellite.DistanceSpaceChannel
}

func (satellite *Satellite) SetDistanceSpaceChannel(channel *DistanceSpaceSatelliteChannel) {
	satellite.DistanceSpaceChannel = channel
}

func (satellite *Satellite) updatePosition() {
	dt := float64(satellite.Dt) * 0.001 // milliseconds to seconds
	satellite.OrbitalAnomaly, satellite.AnomalyElements = satellite.AnomalyCalculations.UpdatePosition(satellite.OrbitalAnomaly, dt)
}

func (satellite *Satellite) nextTimeStep() {
	satellite.TimeStamp += satellite.Dt
}

func (satellite *Satellite) updateSpaceOnDistances() {
	(*satellite.DistanceSpaceChannel) <- UpdateDistancesMessage{
		SatelliteName:    satellite.Name,
		SatelliteAnomaly: satellite.OrbitalAnomaly,
		TimeStamp:        satellite.TimeStamp,
		Distances: satellite.AnomalyCalculations.FindSatellitesInRange(satellite.Id, satellite.OrbitalAnomaly, satellite.AnomalyElements,
			satellite.Orbit.GetOrbitNumber(), float64(satellite.TimeStamp)*0.001),
	}
}

func NewSatellite(id int, orbitalPhase float64, dt int, totalSimulationTime int, orbit helpers.IOrbit, anomalyCalculations helpers.IAnomalyCalculation) ISatellite {
	var newSatellite Satellite

	newSatellite.Id = id
	newSatellite.Name = fmt.Sprintf("%s-%d", orbit.GetOrbitId(), id)
	newSatellite.Dt = dt
	newSatellite.TotalSimulationTime = totalSimulationTime
	newSatellite.TimeStamp = 0
	newSatellite.OrbitalAnomaly = orbitalPhase * (math.Pi / 180.0)
	newSatellite.AnomalyCalculations = anomalyCalculations
	newSatellite.Orbit = orbit
	newSatellite.AnomalyElements = helpers.AnomalyElements{
		AnomalySinus:   math.Sin(newSatellite.OrbitalAnomaly),
		AnomalyCosinus: math.Cos(newSatellite.OrbitalAnomaly),
	}

	return &newSatellite
}

func startSatelliteDistances(mySatellite ISatellite) {
	for mySatellite.getTimeStamp() <= mySatellite.getTotalSimulationTime() {
		mySatellite.updateSpaceOnDistances()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
	close(*mySatellite.GetSpaceChannel())
}

func startSatellite(mySatellite ISatellite) {
	mySatellite.loadForwardingTableInMemory()
	for mySatellite.getTimeStamp() <= mySatellite.getTotalSimulationTime() {

	}
}
