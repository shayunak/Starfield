package actors

import (
	"log"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
)

type GroundStation struct {
	// General
	Name                string
	Dt                  int // in milliseconds
	TimeStamp           int // in milliseconds
	TotalSimulationTime int // in milliseconds

	// Geometrical parameters
	Longitude          float64
	Latitude           float64
	HeadPointAnomaly   float64
	HeadPointAscension float64
	HeadPointAnomalyEl helpers.AnomalyElements
	GSCalculation      helpers.IGroundStationCalculation

	// Packet Level Simulation
	EventQueue helpers.PriorityQueue

	// Goroutines and connections, and channels
	GSLInterfaces        connections.INetworkInterface
	DistanceSpaceChannel *DistanceSpaceDeviceChannel
	SpaceChannel         *SpaceSatelliteChannel
}

type IGroundStation interface {
	Run()
	RunDistances()
	getTimeStamp() int
	getTotalSimulationTime() int
	nextTimeStep()
	updatePosition()
	updateSpaceOnDistances()
	SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel)
	GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel
}

func (gs *GroundStation) RunDistances() {
	log.Default().Println("Running ground station (Distance Mode): ", gs.Name)
	go startGSDistances(gs)
}

func (gs *GroundStation) Run() {
	log.Default().Println("Running ground station: ", gs.Name)
	go startGS(gs)
}

func (gs *GroundStation) getTimeStamp() int {
	return gs.TimeStamp
}

func (gs *GroundStation) getTotalSimulationTime() int {
	return gs.TotalSimulationTime
}

func (gs *GroundStation) SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel) {
	gs.DistanceSpaceChannel = channel
}

func (gs *GroundStation) GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel {
	return gs.DistanceSpaceChannel
}

func (gs *GroundStation) nextTimeStep() {
	gs.TimeStamp += gs.Dt
}

func (gs *GroundStation) updatePosition() {
	dt := float64(gs.Dt) * 0.001 // milliseconds to seconds
	gs.HeadPointAscension = gs.GSCalculation.UpdatePosition(gs.HeadPointAscension, dt)
}

func (gs *GroundStation) updateSpaceOnDistances() {
	(*gs.DistanceSpaceChannel) <- UpdateDistancesMessage{
		DeviceName: gs.Name,
		TimeStamp:  gs.TimeStamp,
		Distances: gs.GSCalculation.FindSatellitesInRange(gs.Name, gs.HeadPointAscension, gs.HeadPointAnomalyEl,
			float64(gs.TimeStamp)*0.001),
	}
}

func startGSDistances(myGS IGroundStation) {
	for myGS.getTimeStamp() <= myGS.getTotalSimulationTime() {
		myGS.updateSpaceOnDistances()
		myGS.nextTimeStep()
		myGS.updatePosition()
	}
	close(*myGS.GetDistanceSpaceChannel())
}

func startGS(myGS IGroundStation) {

}

func NewGroundStation(name string, latitude float64, longitude float64, dt int, totalSimulationTime int,
	headPointAnomaly float64, headPointAscension float64, groundStationCalculation helpers.IGroundStationCalculation,
	headPointAnomalyEl helpers.AnomalyElements) IGroundStation {

	var newGS GroundStation

	newGS.Name = name
	newGS.Dt = dt
	newGS.TotalSimulationTime = totalSimulationTime
	newGS.TimeStamp = 0

	// Geo
	newGS.Latitude = latitude
	newGS.Longitude = longitude
	newGS.GSCalculation = groundStationCalculation
	newGS.HeadPointAnomaly = headPointAnomaly
	newGS.HeadPointAscension = headPointAscension
	newGS.HeadPointAnomalyEl = headPointAnomalyEl

	// Channels
	newGS.EventQueue = make(helpers.PriorityQueue, 0)

	return &newGS
}
