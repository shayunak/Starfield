package helpers

import (
	"fmt"
	"math"
)

type AnomalyElements struct {
	AnomalySinus   float64
	AnomalyCosinus float64
}

type DistanceObject struct {
	Distance      float64
	Anomaly       float64
	AscensionDiff string
	A             float64
	B             float64
}

type IAnomalyCalculation interface {
	FindSatellitesInRange(lengthLimitRatio float64, currentAnomaly float64, anomalyEl AnomalyElements, orbitalAscension float64, timeStamp float64) map[string]DistanceObject
	UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements)
	CalculateDistanceBySatelliteId(firstSatelliteId int, firstSatelliteOrbitId int, secondSatelliteId int, secondSatelliteOrbitId int, timeStamp float64) float64
	GetOrbitalCalculations() IOrbitalCalculations
	GetLengthLimitRatio() float64
	calculateSatelliteIdInRange(lengthLimitRatio float64, currentAnomaly float64, orbitCalc OrbitCalc, timeStamp float64, orbit int) map[int]DistanceObject
	CalculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64
	calculatePhase(satelliteId int, orbitId int) float64
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

func (anomalyCalc *AnomalyCalculations) CalculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64 {
	distance_squared_factor := 2 * (orbitCalc.CosinalCoefficient*math.Cos(otherSatelliteAnomaly) -
		orbitCalc.SinalCoefficient*math.Sin(otherSatelliteAnomaly) + 1.0)

	if distance_squared_factor <= 0.0 {
		return 0.0
	}

	return anomalyCalc.Radius * math.Sqrt(distance_squared_factor)
}

func (anomalyCalc *AnomalyCalculations) calculateSatelliteIdInRange(lengthLimitRatio float64, currentAnomaly float64,
	orbitCalc OrbitCalc, timeStamp float64, orbit int) map[int]DistanceObject {
	satellites := make(map[int]DistanceObject)
	orbitalCalcSize := math.Sqrt(math.Pow(orbitCalc.CosinalCoefficient, 2.0) + math.Pow(orbitCalc.SinalCoefficient, 2.0))
	boundedAnomaly := math.Mod(currentAnomaly, 2.0*math.Pi)
	limitTerm := math.Asin(lengthLimitRatio / orbitalCalcSize)
	phaseTerm := math.Atan(orbitCalc.CosinalCoefficient / orbitCalc.SinalCoefficient)

	if orbitCalc.AscensionDiff < math.Pi && boundedAnomaly >= math.Pi {
		phaseTerm += math.Pi
	}

	if orbitCalc.AscensionDiff >= math.Pi && boundedAnomaly < math.Pi {
		phaseTerm += math.Pi
	}

	lowerRange := phaseTerm + limitTerm
	upperRange := math.Pi - limitTerm + phaseTerm

	if boundedAnomaly >= math.Pi {
		lowerRange += math.Pi
		upperRange += math.Pi
	}

	if boundedAnomaly <= lowerRange {
		lowerRange -= math.Pi
		upperRange -= math.Pi
	}

	initialPhaseShift := 0.0
	if anomalyCalc.PhaseDiffEnabled && orbit%2 == 1 {
		initialPhaseShift = anomalyCalc.AnomalyStep / 2.0
	}
	lowerId := int(math.Ceil((lowerRange - initialPhaseShift - timeStamp*anomalyCalc.MeanMotion) / anomalyCalc.AnomalyStep))
	upperId := int(math.Floor((upperRange - initialPhaseShift - timeStamp*anomalyCalc.MeanMotion) / anomalyCalc.AnomalyStep))

	for i := lowerId; i <= upperId; i++ {
		realAnomaly := float64(i)*anomalyCalc.AnomalyStep + initialPhaseShift + timeStamp*anomalyCalc.MeanMotion
		realId := (i + anomalyCalc.NumberOfSatellitesPerOrbit) % anomalyCalc.NumberOfSatellitesPerOrbit
		satellites[realId] = DistanceObject{
			Anomaly:       realAnomaly,
			Distance:      anomalyCalc.CalculateDistance(orbitCalc, realAnomaly),
			AscensionDiff: orbitCalc.OrbitalRange,
			A:             float64(lengthLimitRatio),
			B:             float64(orbitalCalcSize),
		}
		//satellites[realId] = anomalyCalc.CalculateDistance(orbitCalc, realAnomaly)
	}

	return satellites
}

func (anomalyCalc *AnomalyCalculations) FindSatellitesInRange(lengthLimitRatio float64, currentAnomaly float64,
	anomalyEl AnomalyElements, orbitalAscension float64, timeStamp float64) map[string]DistanceObject {

	satellitesDistances := make(map[string]DistanceObject)
	orbitsInRange := anomalyCalc.OrbitalCalculations.FindOrbitsInRange(lengthLimitRatio, currentAnomaly, anomalyEl, orbitalAscension)
	for orbit, orbitCalc := range orbitsInRange {
		satellites := anomalyCalc.calculateSatelliteIdInRange(lengthLimitRatio, currentAnomaly, orbitCalc, timeStamp, orbit)
		for id, distance := range satellites {
			sat_name := fmt.Sprintf("%s-%d-%d", anomalyCalc.ConsellationName, orbit, id)
			satellitesDistances[sat_name] = distance
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

func (anomalyCalc *AnomalyCalculations) calculatePhase(satelliteId int, orbitId int) float64 {

	phase := float64(satelliteId) * anomalyCalc.AnomalyStep
	if anomalyCalc.PhaseDiffEnabled && orbitId%2 == 1 {
		phase += anomalyCalc.AnomalyStep / 2.0
	}

	return phase
}

// Timestamp shpuld be in seconds
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
