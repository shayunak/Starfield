package helpers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Orbit struct {
	Id                 int
	ConsellationName   string
	Radius             float64 // in meters
	EarthRotaionMotion float64 // in radians per second
	Altitude           float64 // in meters
	Ascension          float64 // in radians
	Inclination        float64 // in radians
	OrbitPhaseDiff     float64 // in radians
}

type IOrbit interface {
	GetOrbitNumber() int
	GetOrbitId() string
	GetConstellationName() string
	GetOrbitName() string
	GetAscension() float64
	GetInclination() float64
	GetRadius() float64
	GetEarthRotaionMotion() float64
	GetAltitude() float64
	IsOwnerSatellite(ownerId string) bool
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

func (orbit *Orbit) GetConstellationName() string {
	return orbit.ConsellationName
}

func (orbit *Orbit) GetAscension() float64 {
	return orbit.Ascension
}

func (orbit *Orbit) GetInclination() float64 {
	return orbit.Inclination
}

func (orbit *Orbit) GetRadius() float64 {
	return orbit.Radius
}

func (orbit *Orbit) GetEarthRotaionMotion() float64 {
	return orbit.EarthRotaionMotion
}

func (orbit *Orbit) GetAltitude() float64 {
	return orbit.Altitude
}

func NewOrbit(radius float64, earthRotationMotion float64, altitude float64, ascension float64, inclination float64, id int,
	consellationName string, phaseDiff float64) IOrbit {
	var newOrbit Orbit

	ascensionRadians := ascension * (math.Pi / 180.0)
	phaseDiffRadians := phaseDiff * (math.Pi / 180.0)

	newOrbit = Orbit{
		Id:                 id,
		ConsellationName:   consellationName,
		Radius:             radius,
		EarthRotaionMotion: earthRotationMotion,
		Altitude:           altitude,
		Ascension:          ascensionRadians,
		Inclination:        inclination,
		OrbitPhaseDiff:     phaseDiffRadians,
	}

	return &newOrbit
}

func (orbit *Orbit) IsOwnerSatellite(ownerId string) bool {
	splittedId := strings.Split(ownerId, "-")
	if len(splittedId) == 3 && splittedId[0] == orbit.GetConstellationName() {
		return true
	}
	return false
}

func GetOrbitAndSatelliteId(satelliteName string) (int, int) {
	splitted := strings.Split(satelliteName, "-")
	orbit, _ := strconv.Atoi(splitted[1])
	id, _ := strconv.Atoi(splitted[2])

	return orbit, id
}
