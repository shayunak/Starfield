package helpers

import (
	"fmt"
	"math"
)

type AnomalyElements struct {
	AnomalySinus   float64
	AnomalyCosinus float64
}

/*type DistanceObject struct {
	Distance      float64
	Anomaly       float64
	AscensionDiff float64
	A             float64
	B             float64
}*/

type IAnomalyCalculation interface {
	FindSatellitesInRange(currentAnomaly float64, anomalyEl AnomalyElements, orbitId int, timeStamp float64) map[string]float64
	UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements)
	calculateSatelliteIdInRange(currentAnomaly float64, orbitCalc OrbitCalc, timeStamp float64, orbit int) map[int]float64
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
	PhaseDiffEnabled           bool
}

func (anomalyCalc *AnomalyCalculations) calculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64 {
	return anomalyCalc.Radius * math.Sqrt(2*(orbitCalc.CosinalCoefficient*math.Cos(otherSatelliteAnomaly)-
		orbitCalc.SinalCoefficient*math.Sin(otherSatelliteAnomaly)+1.0))
}

func (anomalyCalc *AnomalyCalculations) calculateSatelliteIdInRange(currentAnomaly float64, orbitCalc OrbitCalc, timeStamp float64, orbit int) map[int]float64 {
	satellites := make(map[int]float64)
	orbitalCalcSize := math.Sqrt(math.Pow(orbitCalc.CosinalCoefficient, 2) + math.Pow(orbitCalc.SinalCoefficient, 2))
	limitTerm := math.Asin(anomalyCalc.LengthLimitRatio / orbitalCalcSize)
	phaseTerm := math.Atan(orbitCalc.CosinalCoefficient / orbitCalc.SinalCoefficient)

	lowerRange := phaseTerm + limitTerm
	upperRange := math.Pi - limitTerm + phaseTerm

	if currentAnomaly >= upperRange {
		lowerRange += math.Pi
		upperRange += math.Pi
	}

	if currentAnomaly <= lowerRange {
		lowerRange -= math.Pi
		upperRange -= math.Pi
	}

	//log.Default().Printf(fmt.Sprintf("currentAnomaly: %f, Orbit: %d, lowerRange: %f, upperRange: %f", currentAnomaly, orbit, lowerRange, upperRange))

	initialPhaseShift := 0.0
	if anomalyCalc.PhaseDiffEnabled && orbit%2 == 1 {
		initialPhaseShift = anomalyCalc.AnomalyStep / 2.0
	}
	lowerId := int(math.Ceil((lowerRange - initialPhaseShift - timeStamp*anomalyCalc.MeanMotion) / anomalyCalc.AnomalyStep))
	upperId := int(math.Floor((upperRange - initialPhaseShift - timeStamp*anomalyCalc.MeanMotion) / anomalyCalc.AnomalyStep))

	for i := lowerId; i <= upperId; i++ {
		realAnomaly := float64(i)*anomalyCalc.AnomalyStep + initialPhaseShift + timeStamp*anomalyCalc.MeanMotion
		readId := (i + anomalyCalc.NumberOfSatellitesPerOrbit) % anomalyCalc.NumberOfSatellitesPerOrbit
		/*satellites[readId] = DistanceObject{
			Anomaly:       realAnomaly,
			Distance:      anomalyCalc.calculateDistance(orbitCalc, realAnomaly),
			AscensionDiff: orbitCalc.AscensionDiff,
			A:             orbitCalc.CosinalCoefficient,
			B:             orbitCalc.SinalCoefficient,
		}*/
		satellites[readId] = anomalyCalc.calculateDistance(orbitCalc, realAnomaly)
	}

	return satellites
}

func (anomalyCalc *AnomalyCalculations) FindSatellitesInRange(currentAnomaly float64, anomalyEl AnomalyElements, orbitId int, timeStamp float64) map[string]float64 {

	satellitesDistances := make(map[string]float64)
	orbitsInRange := anomalyCalc.OrbitalCalculations.FindOrbitsInRange(anomalyEl, orbitId)
	for orbit, orbitCalc := range orbitsInRange {
		//satellitesDistances[fmt.Sprintf("%s-%d", anomalyCalc.ConsellationName, orbit)] = orbitCalc.CosinalCoefficient
		satellites := anomalyCalc.calculateSatelliteIdInRange(currentAnomaly, orbitCalc, timeStamp, orbit)
		for id, distanceObject := range satellites {
			sat_name := fmt.Sprintf("%s-%d-%d", anomalyCalc.ConsellationName, orbit, id)
			satellitesDistances[sat_name] = distanceObject
		}
	}
	return satellitesDistances
}

func (anomalyCalc *AnomalyCalculations) UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements) {
	newOrbitalAnomaly := prevAnomaly + anomalyCalc.MeanMotion*timeStep
	newAnomalyElements := AnomalyElements{
		AnomalySinus:   math.Sin(newOrbitalAnomaly),
		AnomalyCosinus: math.Cos(newOrbitalAnomaly),
	}
	return newOrbitalAnomaly, newAnomalyElements
}
