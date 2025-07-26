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
	AscensionDiff string
	A             float64
	B             float64
}*/

type IAnomalyCalculation interface {
	FindSatellitesInRange(Id string, lengthLimitRatio float64, anomalyEl AnomalyElements, orbitalAscension float64, timeStamp float64) map[string]float64
	UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements)
	CalculateDistanceBySatelliteId(firstSatelliteId int, firstSatelliteOrbitId int, secondSatelliteId int, secondSatelliteOrbitId int, timeStamp float64) float64
	GetOrbitalCalculations() IOrbitalCalculations
	GetLengthLimitRatio() float64
	GetMaxDistance() float64
	findDistanceforSatelliteId(i int, baseId string, orbit int, timeStamp float64, orbitCalc OrbitCalc, initialPhaseShift float64, satelliteIds *[]string, satelliteDistances *[]float64)
	calculateSatelliteIdInRange(orbit int, orbitCalc OrbitCalc, baseId string, lengthLimitRatio float64, timeStamp float64, satelliteIds *[]string, satelliteDistances *[]float64)
	CalculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64
	calculatePhase(satelliteId int, orbitId int) float64
}

type AnomalyCalculations struct {
	ConsellationName           string
	LengthLimitRatio           float64
	MaxDistance                float64
	NumberOfSatellitesPerOrbit int
	AnomalyStep                float64 // in radians
	MeanMotion                 float64 // in radians per second
	Radius                     float64 // in meters
	OrbitalCalculations        IOrbitalCalculations
	PhaseDiffEnabled           bool
}

// Cuda Compatible
func (anomalyCalc *AnomalyCalculations) CalculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64 {
	distance_squared_factor := 2 * (orbitCalc.CosinalCoefficient*math.Cos(otherSatelliteAnomaly) -
		orbitCalc.SinalCoefficient*math.Sin(otherSatelliteAnomaly) + 1.0)

	if distance_squared_factor <= 0.0 {
		return 0.0
	}

	return anomalyCalc.Radius * math.Sqrt(distance_squared_factor)
}

func (anomalyCalc *AnomalyCalculations) findDistanceforSatelliteId(i int, baseId string, orbit int, timeStamp float64, orbitCalc OrbitCalc,
	initialPhaseShift float64, satelliteIds *[]string, satelliteDistances *[]float64) {
	realAnomaly := float64(i)*anomalyCalc.AnomalyStep + initialPhaseShift + timeStamp*anomalyCalc.MeanMotion
	realId := (i + anomalyCalc.NumberOfSatellitesPerOrbit) % anomalyCalc.NumberOfSatellitesPerOrbit
	satelliteId := fmt.Sprintf("%s-%d-%d", anomalyCalc.ConsellationName, orbit, realId)
	if baseId != satelliteId {
		*satelliteIds = append(*satelliteIds, satelliteId)
		*satelliteDistances = append(*satelliteDistances, anomalyCalc.CalculateDistance(orbitCalc, realAnomaly))
	}
}

// Cuda Compatible
func (anomalyCalc *AnomalyCalculations) calculateSatelliteIdInRange(orbit int, orbitCalc OrbitCalc, baseId string,
	lengthLimitRatio float64, timeStamp float64, satelliteIds *[]string, satelliteDistances *[]float64) {
	orbitalCalcSize := math.Sqrt(math.Pow(orbitCalc.CosinalCoefficient, 2.0) + math.Pow(orbitCalc.SinalCoefficient, 2.0))
	limitTerm := math.Asin(lengthLimitRatio / orbitalCalcSize)
	phaseTerm := math.Atan2(orbitCalc.CosinalCoefficient, orbitCalc.SinalCoefficient)

	lowerRange := phaseTerm + limitTerm
	upperRange := math.Pi - limitTerm + phaseTerm

	initialPhaseShift := 0.0
	if anomalyCalc.PhaseDiffEnabled && orbit%2 == 1 {
		initialPhaseShift = anomalyCalc.AnomalyStep / 2.0
	}

	lowerId := int(math.Ceil((lowerRange - initialPhaseShift - timeStamp*anomalyCalc.MeanMotion) / anomalyCalc.AnomalyStep))
	upperId := int(math.Floor((upperRange - initialPhaseShift - timeStamp*anomalyCalc.MeanMotion) / anomalyCalc.AnomalyStep))

	for i := lowerId; i <= upperId; i++ {
		anomalyCalc.findDistanceforSatelliteId(i, baseId, orbit, timeStamp, orbitCalc, initialPhaseShift, satelliteIds, satelliteDistances)
	}
}

func (anomalyCalc *AnomalyCalculations) FindSatellitesInRange(Id string, lengthLimitRatio float64, anomalyEl AnomalyElements,
	orbitalAscension float64, timeStamp float64) map[string]float64 {

	var satelliteDistances []float64
	var satelliteIds []string
	orbitsInRange := anomalyCalc.OrbitalCalculations.FindOrbitsInRange(lengthLimitRatio, anomalyEl, orbitalAscension)
	for orbit, orbitCalc := range orbitsInRange {
		anomalyCalc.calculateSatelliteIdInRange(orbit, orbitCalc, Id, lengthLimitRatio, timeStamp, &satelliteIds, &satelliteDistances)
	}
	return zip_satellite_ids_with_distances(satelliteIds, satelliteDistances)
}

// Cuda Compatible
func (anomalyCalc *AnomalyCalculations) UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements) {
	newOrbitalAnomaly := prevAnomaly + anomalyCalc.MeanMotion*timeStep
	newAnomalyElements := AnomalyElements{
		AnomalySinus:   math.Sin(newOrbitalAnomaly),
		AnomalyCosinus: math.Cos(newOrbitalAnomaly),
	}
	return newOrbitalAnomaly, newAnomalyElements
}

// Cuda Compatible
func (anomalyCalc *AnomalyCalculations) calculatePhase(satelliteId int, orbitId int) float64 {
	phase := float64(satelliteId) * anomalyCalc.AnomalyStep
	if anomalyCalc.PhaseDiffEnabled && orbitId%2 == 1 {
		phase += anomalyCalc.AnomalyStep / 2.0
	}

	return phase
}

// Cuda Compatible
// Timestamp should be in seconds
func (anomalyCalc *AnomalyCalculations) CalculateDistanceBySatelliteId(firstSatelliteId int, firstSatelliteOrbitId int,
	secondSatelliteId int, secondSatelliteOrbitId int, timeStamp float64) float64 {

	firstPhase := anomalyCalc.calculatePhase(firstSatelliteId, firstSatelliteOrbitId)
	secondPhase := anomalyCalc.calculatePhase(secondSatelliteId, secondSatelliteOrbitId)
	phaseDiff := firstPhase - secondPhase
	ascensionDiff := float64(firstSatelliteOrbitId-secondSatelliteOrbitId) * anomalyCalc.OrbitalCalculations.GetAscensionStep()
	ascensionDiffSinus := math.Sin(ascensionDiff)
	ascensionDiffCosinus := math.Cos(ascensionDiff)
	phaseDiffSinus := math.Sin(phaseDiff)
	phaseDiffCosinus := math.Cos(phaseDiff)
	inclinationSinus := anomalyCalc.OrbitalCalculations.GetInclinationSinus()
	inclinationCosinus := anomalyCalc.OrbitalCalculations.GetInclinationCosinus()
	timeTermCosinus := math.Cos(2.0*anomalyCalc.MeanMotion*timeStamp + firstPhase + secondPhase)

	phaseDiffSinusTerm := 2.0 * inclinationCosinus * ascensionDiffSinus * phaseDiffSinus
	phaseDiffCosinusTerm := ((1+math.Pow(inclinationCosinus, 2.0))*ascensionDiffCosinus + math.Pow(inclinationSinus, 2.0)) * phaseDiffCosinus
	timeTerm := (1 - ascensionDiffCosinus) * math.Pow(inclinationSinus, 2.0) * timeTermCosinus

	return anomalyCalc.Radius * math.Sqrt(2.0+phaseDiffSinusTerm-phaseDiffCosinusTerm+timeTerm)
}

func (anomalyCalc *AnomalyCalculations) GetOrbitalCalculations() IOrbitalCalculations {
	return anomalyCalc.OrbitalCalculations
}

func (anomalyCalc *AnomalyCalculations) GetLengthLimitRatio() float64 {
	return anomalyCalc.LengthLimitRatio
}

func (anomalyCalc *AnomalyCalculations) GetMaxDistance() float64 {
	return anomalyCalc.MaxDistance
}

func zip_satellite_ids_with_distances(inRangeIds []string, distances []float64) map[string]float64 {
	distancesMap := make(map[string]float64)
	for i, id := range inRangeIds {
		distancesMap[id] = distances[i]
	}
	return distancesMap
}
