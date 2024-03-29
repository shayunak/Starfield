package actors

import (
	"fmt"
	"log"
	"math"
)

type CartesianCoordinates struct {
	X float64 // in meters
	Y float64 // in meters
	Z float64 // in meters
}

type Orbit struct {
	Radius      float64 // in meters
	Altitude    float64 // in meters
	Ascension   float64 // in radians
	Inclination float64 // in radians
}

type Satellite struct {
	Id             string
	Position       CartesianCoordinates
	Orbit          Orbit
	SpaceChannel   *SpaceSatelliteChannel
	Dt             int     // in milliseconds
	TimeStamp      int     // in milliseconds
	OrbitalMotion  float64 // in radians per second
	OrbitalAnomaly float64 // in radians
}

type ISatellite interface {
	Run()
	GetSpaceChannel() *SpaceSatelliteChannel
	updatePosition()
	updateSpaceOnPosition()
	nextTimeStep()
	checkChannelLiveness() bool
}

type IOrbit interface {
	convertToCartesian(anomaly float64) CartesianCoordinates
}

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Id)
	go startSatellite(satellite)
}

func (satellite *Satellite) GetSpaceChannel() *SpaceSatelliteChannel {
	return satellite.SpaceChannel
}

func (satellite *Satellite) updatePosition() {
	satellite.OrbitalAnomaly += satellite.OrbitalMotion * float64(satellite.Dt) * 0.001 // milliseconds to seconds
	satellite.Position = satellite.Orbit.convertToCartesian(satellite.OrbitalAnomaly)
}

func (satellite *Satellite) nextTimeStep() {
	satellite.TimeStamp += satellite.Dt
}

func (satellite *Satellite) updateSpaceOnPosition() {
	(*satellite.SpaceChannel) <- UpdatePoisitionMessage{
		SatelliteId: satellite.Id,
		OrbitId:     satellite.Orbit.GetOrbitId(),
		Anomaly:     satellite.OrbitalAnomaly,
		Position:    satellite.Position,
		TimeStamp:   satellite.TimeStamp,
	}
}

func (satellite *Satellite) checkChannelLiveness() bool {
	_, ok := <-(*satellite.SpaceChannel)
	return ok
}

func (orbit Orbit) convertToCartesian(anomaly float64) CartesianCoordinates {
	return CartesianCoordinates{
		X: orbit.Radius * (math.Cos(anomaly)*math.Cos(orbit.Ascension) -
			math.Sin(anomaly)*math.Cos(orbit.Inclination)*math.Sin(orbit.Ascension)),
		Y: orbit.Radius * (math.Cos(anomaly)*math.Sin(orbit.Ascension) +
			math.Sin(anomaly)*math.Cos(orbit.Inclination)*math.Cos(orbit.Ascension)),
		Z: orbit.Radius * math.Sin(anomaly) * math.Sin(orbit.Inclination),
	}
}

func (orbit Orbit) GetOrbitId() string {
	altitudeKM := int(orbit.Altitude * 0.001)
	ascensionDegree := int(orbit.Ascension * 180.0 / math.Pi)
	inclinationDegree := int(orbit.Inclination * 180.0 / math.Pi)

	return fmt.Sprintf("%dkm-%d'-%d'", altitudeKM, ascensionDegree, inclinationDegree)
}

func startSatellite(mySatellite ISatellite) {
	for {
		mySatellite.updateSpaceOnPosition()
		if !mySatellite.checkChannelLiveness() {
			break
		}
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
}

func NewSatellite(id string, orbitAltitude float64, earthRadius float64, orbitalMotionRevPerDay float64,
	orbitalPhase float64, orbitInclination float64, orbitAscension float64, dt int) ISatellite {
	var newSatellite Satellite

	spaceChannel := make(SpaceSatelliteChannel)

	newSatellite.Id = id
	newSatellite.Dt = dt
	newSatellite.TimeStamp = 0
	newSatellite.SpaceChannel = &spaceChannel
	newSatellite.OrbitalMotion = orbitalMotionRevPerDay * ((2.0 * math.Pi) / (24.0 * 60.0 * 60.0))
	newSatellite.OrbitalAnomaly = orbitalPhase * (math.Pi / 180.0)
	newSatellite.Orbit = Orbit{
		Radius:      orbitAltitude + earthRadius,
		Altitude:    orbitAltitude,
		Ascension:   orbitAscension * (math.Pi / 180.0),
		Inclination: orbitInclination * (math.Pi / 180.0),
	}
	newSatellite.Position = newSatellite.Orbit.convertToCartesian(newSatellite.OrbitalAnomaly)

	return &newSatellite
}
