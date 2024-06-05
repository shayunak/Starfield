package helpers

import (
	"math"
)

type OrbitalCalculations struct {
	InclinationSinus   float64
	InclinationCosinus float64
	LengthLimitRatio   float64
	NumberOfOrbits     int
	AscensionStep      float64 // in radians
	MinAscensionAngle  float64 // in radians
	MaxAscensionAngle  float64 // in radians
}

type OrbitCalc struct {
	CosinalCoefficient float64
	SinalCoefficient   float64
	AscensionDiff      float64
}

type IOrbitalCalculations interface {
	FindOrbitsInRange(currentAnomaly float64, anomalyEl AnomalyElements, orbit_id int) map[int]OrbitCalc
	isOrbitValid(orbitId int) bool
	getRealOrbitId(id int, orbitId int) (int, float64)
}

func calculateLimits(lengthLimitRatio float64, inclinationSinus float64, inclinationCosinus float64,
	anomalySinus float64) (float64, float64) {

	ISLLengthLimit := math.Sqrt(1 - math.Pow(lengthLimitRatio, 2))
	baseTrig := inclinationSinus * inclinationCosinus * anomalySinus
	denominator := inclinationSinus * math.Sqrt(1.0-math.Pow(anomalySinus*inclinationSinus, 2))
	lowerLimit := (baseTrig - ISLLengthLimit) / denominator
	upperLimit := (baseTrig + ISLLengthLimit) / denominator

	return math.Asin(math.Max(lowerLimit, -1.0)), math.Asin(math.Min(upperLimit, 1.0))
}

func calculateCosinalCoefficient(inclinationCosinus float64, anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	CosinalMultplication := anomalyEl.AnomalyCosinus * math.Cos(ascensionDiff)
	SinalMultiplication := anomalyEl.AnomalySinus * math.Sin(ascensionDiff) * inclinationCosinus

	return SinalMultiplication - CosinalMultplication
}

func calculateSinalCoefficient(inclinationCosinus float64, inclinationSinus float64, anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	CosinalMultplication := anomalyEl.AnomalyCosinus * math.Sin(ascensionDiff) * inclinationCosinus
	SinalMultiplication := anomalyEl.AnomalySinus * (math.Pow(inclinationCosinus, 2)*math.Cos(ascensionDiff) + math.Pow(inclinationSinus, 2))

	return CosinalMultplication + SinalMultiplication
}

func (orbitalCalc *OrbitalCalculations) isOrbitValid(id int) bool {
	ascensionCalculated := math.Mod(orbitalCalc.AscensionStep*float64(id)+orbitalCalc.MinAscensionAngle+2*math.Pi, 2*math.Pi)
	return ascensionCalculated >= orbitalCalc.MinAscensionAngle && ascensionCalculated < orbitalCalc.MaxAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) getRealOrbitId(id int, orbitId int) (int, float64) {
	ascensionCalculated := math.Mod(orbitalCalc.AscensionStep*float64(id)+orbitalCalc.MinAscensionAngle+2*math.Pi, 2*math.Pi)
	realId := int((ascensionCalculated - orbitalCalc.MinAscensionAngle) / orbitalCalc.AscensionStep)
	ascensionDiff := float64(orbitId-realId) * orbitalCalc.AscensionStep
	return realId, ascensionDiff
}

func (orbitalCalc *OrbitalCalculations) FindOrbitsInRange(currentAnomaly float64, anomalyEl AnomalyElements, orbitId int) map[int]OrbitCalc {
	inRangeOrbits := make(map[int]OrbitCalc)

	boundedAnomaly := math.Mod(currentAnomaly, 2*math.Pi)
	LD, LU := calculateLimits(orbitalCalc.LengthLimitRatio, orbitalCalc.InclinationSinus,
		orbitalCalc.InclinationCosinus, anomalyEl.AnomalySinus)

	Phi := math.Atan(orbitalCalc.InclinationCosinus * anomalyEl.AnomalySinus / anomalyEl.AnomalyCosinus)
	firstRangeMin := int(math.Ceil((Phi - LU) / orbitalCalc.AscensionStep))
	firstRangeMax := int(math.Floor((Phi - LD) / orbitalCalc.AscensionStep))
	secondRangeMin := int(math.Ceil((Phi + LD - math.Pi) / orbitalCalc.AscensionStep))
	secondRangeMax := int(math.Floor((Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

	if boundedAnomaly > math.Pi/2.0 && boundedAnomaly <= 3.0*math.Pi/2.0 {
		firstRangeMin = int(math.Ceil((Phi - LU + math.Pi) / orbitalCalc.AscensionStep))
		firstRangeMax = int(math.Floor((Phi - LD + math.Pi) / orbitalCalc.AscensionStep))
		secondRangeMin = int(math.Ceil((Phi + LD) / orbitalCalc.AscensionStep))
		secondRangeMax = int(math.Floor((Phi + LU) / orbitalCalc.AscensionStep))
	}

	// Calculate First Range
	for i := firstRangeMin; i <= firstRangeMax; i++ {
		if orbitalCalc.isOrbitValid(orbitId + i) {
			id, ascensionDiff := orbitalCalc.getRealOrbitId(orbitId+i, orbitId)
			inRangeOrbits[id] = OrbitCalc{
				CosinalCoefficient: calculateCosinalCoefficient(orbitalCalc.InclinationCosinus, anomalyEl, ascensionDiff),
				SinalCoefficient:   calculateSinalCoefficient(orbitalCalc.InclinationCosinus, orbitalCalc.InclinationSinus, anomalyEl, ascensionDiff),
				AscensionDiff:      ascensionDiff,
			}
		}
	}

	// Calculate Second Range
	for i := secondRangeMin; i <= secondRangeMax; i++ {
		if orbitalCalc.isOrbitValid(orbitId + i) {
			id, ascensionDiff := orbitalCalc.getRealOrbitId(orbitId+i, orbitId)
			inRangeOrbits[id] = OrbitCalc{
				CosinalCoefficient: calculateCosinalCoefficient(orbitalCalc.InclinationCosinus, anomalyEl, ascensionDiff),
				SinalCoefficient:   calculateSinalCoefficient(orbitalCalc.InclinationCosinus, orbitalCalc.InclinationSinus, anomalyEl, ascensionDiff),
				AscensionDiff:      ascensionDiff,
			}
		}
	}

	return inRangeOrbits
}
