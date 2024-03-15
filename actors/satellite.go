package actors

import "math"

type CartesianCoordinates struct {
	X float64 // in meters
	Y float64 // in meters
	Z float64 // in meters
}

type Orbit struct {
	Radius      float64 // in meters
	Ascension   float64 // in radians
	Inclination float64 // in radians
}

type Satellite struct {
	Id             string
	Position       CartesianCoordinates
	Orbit          Orbit
	OrbitalMotion  float64 // in radians per second
	OrbitalAnomaly float64 // in radians
}

type ISatellite interface {
	updatePosition(dt float64)
}

type IOrbit interface {
	convertToCartesian(anomaly float64) CartesianCoordinates
}

func (satellite Satellite) updatePosition(dt float64) {
	satellite.OrbitalAnomaly += satellite.OrbitalMotion * dt
}

func (orbit Orbit) convertToCartesian(anomaly float64) CartesianCoordinates {
	return CartesianCoordinates{0.0, 0.0, 0.0}
}

func NewSatellite(id string, orbitAltitude float64, earthRadius float64, orbitalMotionRevPerDay float64,
	orbitalPhase float64, orbitInclination float64, orbitAscension float64) ISatellite {
	var newSatellite Satellite

	newSatellite.Id = id
	newSatellite.OrbitalMotion = orbitalMotionRevPerDay * ((2.0 * math.Pi) / (24.0 * 60.0 * 60.0))
	newSatellite.OrbitalAnomaly = orbitalPhase * (math.Pi / 180.0)
	newSatellite.Orbit = Orbit{
		orbitAltitude + earthRadius,
		orbitAscension * (math.Pi / 180.0),
		orbitInclination * (math.Pi / 180.0),
	}
	newSatellite.Position = newSatellite.Orbit.convertToCartesian(newSatellite.OrbitalAnomaly)

	return newSatellite
}
