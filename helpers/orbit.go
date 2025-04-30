package helpers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type GroundStationEntry struct {
	Name       string
	CalcValues OrbitCalc
}

type CartesianCoordinates struct {
	X float64 // in meters
	Y float64 // in meters
	Z float64 // in meters
}

type Orbit struct {
	Id                          int
	ConsellationName            string
	Radius                      float64             // in meters
	EarthRotaionMotion          float64             // in radians per second
	Altitude                    float64             // in meters
	Ascension                   float64             // in radians
	Inclination                 float64             // in radians
	OrbitPhaseDiff              float64             // in radians
	GroundStations              *GroundStationSpecs // ground stations covered by the orbit
	GroundStationsDistanceLimit float64
}

type IOrbit interface {
	GetOrbitNumber() int
	GetOrbitId() string
	GetOrbitName() string
	GetAscension() float64
	ConvertToCartesian(anomaly float64) CartesianCoordinates
	GetCoveringGroundStations(timeStamp float64, anomaly float64, anomalyCalculation IAnomalyCalculation) map[string]float64
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

func (orbit *Orbit) GetAscension() float64 {
	return orbit.Ascension
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

func NewOrbit(radius float64, earthRotationMotion float64, altitude float64, ascension float64, inclination float64, id int,
	consellationName string, phaseDiff float64, groundStationSpecs *GroundStationSpecs, groundStationDistanceLimit float64) IOrbit {
	var newOrbit Orbit

	ascensionRadians := ascension * (math.Pi / 180.0)
	phaseDiffRadians := phaseDiff * (math.Pi / 180.0)

	newOrbit = Orbit{
		Id:                          id,
		ConsellationName:            consellationName,
		Radius:                      radius,
		EarthRotaionMotion:          earthRotationMotion,
		Altitude:                    altitude,
		Ascension:                   ascensionRadians,
		Inclination:                 inclination,
		OrbitPhaseDiff:              phaseDiffRadians,
		GroundStations:              groundStationSpecs,
		GroundStationsDistanceLimit: groundStationDistanceLimit,
	}

	return &newOrbit
}

func (orbit *Orbit) calculateGSDistance(headPointAnomalyEl AnomalyElements, headPointAscension float64, anomaly float64, anomalyCalculation IAnomalyCalculation) float64 {
	orbitalCalculations := anomalyCalculation.GetOrbitalCalculations()
	ascensionDiff := headPointAscension - orbit.Ascension

	orbitalCalc := OrbitCalc{
		CosinalCoefficient: orbitalCalculations.calculateCosinalCoefficient(headPointAnomalyEl, ascensionDiff),
		SinalCoefficient:   orbitalCalculations.calculateSinalCoefficient(headPointAnomalyEl, ascensionDiff),
		AscensionDiff:      ascensionDiff,
	}

	return anomalyCalculation.CalculateDistance(orbitalCalc, anomaly)
}

func (orbit *Orbit) GetCoveringGroundStations(timeStamp float64, anomaly float64, anomalyCalculation IAnomalyCalculation) map[string]float64 {
	distances := make(map[string]float64)
	earthOrbitRatio := 1.0 - orbit.Altitude/orbit.Radius

	for gsName, gsSpec := range *orbit.GroundStations {
		gsAscension := gsSpec.HeadPointAscension + orbit.EarthRotaionMotion*timeStamp
		distance := orbit.calculateGSDistance(gsSpec.HeadPointAnomalyEl, gsAscension, anomaly, anomalyCalculation)

		if distance < orbit.GroundStationsDistanceLimit {
			updatedDistance := math.Sqrt(math.Pow(orbit.Altitude, 2.0) + earthOrbitRatio*math.Pow(distance, 2.0))
			distances[gsName] = updatedDistance
		}
	}
	return distances
}

func GetOrbitAndSatelliteId(satelliteName string) (int, int) {
	splitted := strings.Split(satelliteName, "-")
	orbit, _ := strconv.Atoi(splitted[1])
	id, _ := strconv.Atoi(splitted[2])

	return orbit, id
}
