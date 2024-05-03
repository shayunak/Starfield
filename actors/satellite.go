package actors

import (
	"fmt"
	"log"
	"math"

	"github.com/shayunak/SatSimGo/helpers"
)

type Satellite struct {
	Name string
	Id   int
	// Position            helpers.CartesianCoordinates (Unnecessary for satellite distances calculations)
	AnomalyElements     helpers.AnomalyElements
	Orbit               helpers.IOrbit
	SpaceChannel        *SpaceSatelliteChannel
	Dt                  int     // in milliseconds
	TimeStamp           int     // in milliseconds
	OrbitalAnomaly      float64 // in radians
	AnomalyCalculations helpers.IAnomalyCalculation
}

type ISatellite interface {
	Run()
	GetSpaceChannel() *SpaceSatelliteChannel
	updatePosition()
	updateSpaceOnDistances()
	nextTimeStep()
	checkChannelLiveness() bool
}

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Id)
	go startSatellite(satellite)
}

func (satellite *Satellite) GetSpaceChannel() *SpaceSatelliteChannel {
	return satellite.SpaceChannel
}

func (satellite *Satellite) updatePosition() {
	dt := float64(satellite.Dt) * 0.001 // milliseconds to seconds
	satellite.OrbitalAnomaly, satellite.AnomalyElements = satellite.AnomalyCalculations.UpdatePosition(satellite.OrbitalAnomaly, dt)
}

func (satellite *Satellite) nextTimeStep() {
	satellite.TimeStamp += satellite.Dt
}

func (satellite *Satellite) updateSpaceOnDistances() {
	(*satellite.SpaceChannel) <- UpdateDistancesMessage{
		SatelliteName: satellite.Name,
		TimeStamp:     satellite.TimeStamp,
		Distances: satellite.AnomalyCalculations.FindSatellitesInRange(satellite.AnomalyElements,
			satellite.Orbit.GetOrbitNumber(), float64(satellite.TimeStamp)*0.001),
	}
}

func (satellite *Satellite) checkChannelLiveness() bool {
	_, ok := <-(*satellite.SpaceChannel)
	return ok
}

func startSatellite(mySatellite ISatellite) {
	for {
		mySatellite.updateSpaceOnDistances()
		if !mySatellite.checkChannelLiveness() {
			break
		}
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
}

func NewSatellite(id int, orbitalPhase float64, dt int, orbit helpers.IOrbit, anomalyCalculations helpers.IAnomalyCalculation) ISatellite {
	var newSatellite Satellite

	spaceChannel := make(SpaceSatelliteChannel)

	newSatellite.Id = id
	newSatellite.Name = fmt.Sprintf("%s-%d", orbit.GetOrbitId(), id)
	newSatellite.Dt = dt
	newSatellite.TimeStamp = 0
	newSatellite.SpaceChannel = &spaceChannel
	newSatellite.OrbitalAnomaly = orbitalPhase * (math.Pi / 180.0)
	newSatellite.AnomalyCalculations = anomalyCalculations
	newSatellite.Orbit = orbit
	newSatellite.AnomalyElements = helpers.AnomalyElements{
		AnomalySinus:   math.Sin(orbitalPhase),
		AnomalyCosinus: math.Cos(orbitalPhase),
	}

	return &newSatellite
}
