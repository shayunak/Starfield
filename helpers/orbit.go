package helpers

import (
	"fmt"
	"math"
)

type CartesianCoordinates struct {
	X float64 // in meters
	Y float64 // in meters
	Z float64 // in meters
}

type Orbit struct {
	Id               int
	ConsellationName string
	Radius           float64 // in meters
	Altitude         float64 // in meters
	Ascension        float64 // in radians
	Inclination      float64 // in radians
	OrbitPhaseDiff   float64 // in radians
}

type IOrbit interface {
	GetOrbitNumber() int
	GetOrbitId() string
	GetOrbitName() string
	ConvertToCartesian(anomaly float64) CartesianCoordinates
}

func (orbit *Orbit) GetOrbitName() string {
	altitudeKM := int(orbit.Altitude * 0.001)
	ascensionDegree := int(orbit.Ascension * 180.0 / math.Pi)
	inclinationDegree := int(orbit.Inclination * 180.0 / math.Pi)

	return fmt.Sprintf("%dkm-%d'-%d'", altitudeKM, ascensionDegree, inclinationDegree)
}

func (orbit *Orbit) GetOrbitId() string {
	return fmt.Sprintf("%s-%d", orbit.ConsellationName, orbit.Id)
}

func (orbit *Orbit) GetOrbitNumber() int {
	return orbit.Id
}

func (orbit *Orbit) ConvertToCartesian(anomaly float64) CartesianCoordinates {
	return CartesianCoordinates{
		X: orbit.Radius * (math.Cos(anomaly)*math.Cos(orbit.Ascension) -
			math.Sin(anomaly)*math.Cos(orbit.Inclination)*math.Sin(orbit.Ascension)),
		Y: orbit.Radius * (math.Cos(anomaly)*math.Sin(orbit.Ascension) +
			math.Sin(anomaly)*math.Cos(orbit.Inclination)*math.Cos(orbit.Ascension)),
		Z: orbit.Radius * math.Sin(anomaly) * math.Sin(orbit.Inclination),
	}
}

func NewOrbit(radius float64, altitude float64, ascension float64, inclination float64, id int,
	consellationName string, phaseDiff float64) IOrbit {
	var newOrbit Orbit

	ascensionRadians := ascension * (math.Pi / 180.0)
	phaseDiffRadians := phaseDiff * (math.Pi / 180.0)

	newOrbit = Orbit{
		Id:               id,
		ConsellationName: consellationName,
		Radius:           radius,
		Altitude:         altitude,
		Ascension:        ascensionRadians,
		Inclination:      inclination,
		OrbitPhaseDiff:   phaseDiffRadians,
	}

	return &newOrbit
}
