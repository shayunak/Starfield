package helpers

import (
	"fmt"
	"math"
)

type AnomalyElements struct {
	AnomalySinus   float64
	AnomalyCosinus float64
}

type IAnomalyCalculation interface {
	FindSatellitesInRange(anomalyEl AnomalyElements, orbitId int, timeStamp float64) map[string]float64
	UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements)
	calculateSatelliteIdInRange(orbitCalc OrbitCalc, timeStamp float64, orbit int) map[int]float64
	calculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64
}

type AnomalyCalculations struct {
	ConsellationName           string
	LengthLimitRatio           float64
	NumberOfSatellitesPerOrbit int
	AnomalyStep                float64 // in radians
	MeanMotion                 float64 // in radians per second
	Radius                     float64 // in meters
	OrbitalCalculations        IOrbitalCalculations
}

func (anomalyCalc *AnomalyCalculations) calculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64 {
	return anomalyCalc.Radius * math.Sqrt(2*(orbitCalc.CosinalCoefficient*math.Cos(otherSatelliteAnomaly)+
		orbitCalc.SinalCoefficient*math.Sin(otherSatelliteAnomaly)))
}

func (anomalyCalc *AnomalyCalculations) calculateSatelliteIdInRange(orbitCalc OrbitCalc, timeStamp float64, orbit int) map[int]float64 {

	satellites := make(map[int]float64)
	orbitalCalcSize := math.Sqrt(math.Pow(orbitCalc.CosinalCoefficient, 2) + math.Pow(orbitCalc.SinalCoefficient, 2))
	limitTerm := math.Acos(anomalyCalc.LengthLimitRatio / orbitalCalcSize)
	phaseTerm := math.Atan(orbitCalc.SinalCoefficient / orbitCalc.CosinalCoefficient)
	lowerRange := limitTerm - phaseTerm
	upperRange := 2*math.Pi - limitTerm - phaseTerm
	initialPhaseShift := 0.0
	if orbit%2 == 1 {
		initialPhaseShift = anomalyCalc.AnomalyStep / 2.0
	}
	anomalyChangeInTime := math.Mod(timeStamp*anomalyCalc.MeanMotion, 2*math.Pi)
	lowerId := int(math.Ceil((lowerRange - initialPhaseShift - anomalyChangeInTime) / anomalyCalc.AnomalyStep))
	upperId := int(math.Floor((upperRange - initialPhaseShift - anomalyChangeInTime) / anomalyCalc.AnomalyStep))

	for i := lowerId; i <= upperId; i++ {
		realAnomaly := float64(i)*anomalyCalc.AnomalyStep + initialPhaseShift + anomalyChangeInTime
		readId := (i + anomalyCalc.NumberOfSatellitesPerOrbit) % anomalyCalc.NumberOfSatellitesPerOrbit
		satellites[readId] = anomalyCalc.calculateDistance(orbitCalc, realAnomaly)
	}

	return satellites
}

func (anomalyCalc *AnomalyCalculations) FindSatellitesInRange(anomalyEl AnomalyElements, orbitId int, timeStamp float64) map[string]float64 {

	satellitesDistances := make(map[string]float64)
	orbitsInRange := anomalyCalc.OrbitalCalculations.FindOrbitsInRange(anomalyEl, orbitId)
	for orbit, orbitCalc := range orbitsInRange {
		satellites := anomalyCalc.calculateSatelliteIdInRange(orbitCalc, timeStamp, orbit)
		for id, distance := range satellites {
			sat_name := fmt.Sprintf("%s-%d-%d", anomalyCalc.ConsellationName, orbit, id)
			satellitesDistances[sat_name] = distance
		}
	}
	return satellitesDistances
}

func (anomalyCalc *AnomalyCalculations) UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements) {
	newOrbitalAnomaly := math.Mod(prevAnomaly+anomalyCalc.MeanMotion*timeStep, 2*math.Pi)
	newAnomalyElements := AnomalyElements{
		AnomalySinus:   math.Sin(newOrbitalAnomaly),
		AnomalyCosinus: math.Cos(newOrbitalAnomaly),
	}
	return newOrbitalAnomaly, newAnomalyElements
}
