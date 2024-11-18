package helpers

import (
	"fmt"
	"math"
)

type OrbitalCalculations struct {
	InclinationSinus   float64
	InclinationCosinus float64
	NumberOfOrbits     int
	AscensionStep      float64 // in radians
	MinAscensionAngle  float64 // in radians
	MaxAscensionAngle  float64 // in radians
}

type OrbitCalc struct {
	CosinalCoefficient float64
	SinalCoefficient   float64
	AscensionDiff      float64
	OrbitalRange       string
}

type IOrbitalCalculations interface {
	GetInclinationSinus() float64
	GetInclinationCosinus() float64
	GetAscensionStep() float64
	FindOrbitsInRange(lengthLimitRatio float64, currentAnomaly float64, anomalyEl AnomalyElements, orbitalAscension float64) map[int]OrbitCalc
	IsOrbitAngleValid(angle float64) bool
	getRealOrbitId(ascensionCalculated float64, orbitalAscension float64) (int, float64)
	analyzeOrbitRange(LU float64, LD float64, rangeMin int, rangeMax int, inRangeOrbits *map[int]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements)
	calculateLimits(lengthLimitRatio float64, anomalySinus float64) (float64, float64)
	calculateCosinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64
	calculateSinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64
}

func (orbitalCalc *OrbitalCalculations) GetInclinationSinus() float64 {
	return orbitalCalc.InclinationSinus
}

func (orbitalCalc *OrbitalCalculations) GetInclinationCosinus() float64 {
	return orbitalCalc.InclinationCosinus
}

func (orbitalCalc *OrbitalCalculations) GetAscensionStep() float64 {
	return orbitalCalc.AscensionStep
}

func (orbitalCalc *OrbitalCalculations) ConvertOrbitIdToAscension(orbitId int) float64 {
	return float64(orbitId)*orbitalCalc.AscensionStep + orbitalCalc.MinAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) calculateLimits(lengthLimitRatio float64, anomalySinus float64) (float64, float64) {

	ISLLengthLimit := math.Sqrt(1 - math.Pow(lengthLimitRatio, 2))
	baseTrig := orbitalCalc.InclinationSinus * orbitalCalc.InclinationCosinus * anomalySinus
	denominator := orbitalCalc.InclinationSinus * math.Sqrt(1.0-math.Pow(anomalySinus*orbitalCalc.InclinationSinus, 2))
	lowerLimit := (baseTrig - ISLLengthLimit) / denominator
	upperLimit := (baseTrig + ISLLengthLimit) / denominator

	return math.Asin(math.Max(lowerLimit, -1.0)), math.Asin(math.Min(upperLimit, 1.0))
}

func (orbitalCalc *OrbitalCalculations) calculateCosinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	CosinalMultplication := anomalyEl.AnomalyCosinus * math.Cos(ascensionDiff)
	SinalMultiplication := anomalyEl.AnomalySinus * math.Sin(ascensionDiff) * orbitalCalc.InclinationCosinus

	return SinalMultiplication - CosinalMultplication
}

func (orbitalCalc *OrbitalCalculations) calculateSinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	CosinalMultplication := anomalyEl.AnomalyCosinus * math.Sin(ascensionDiff) * orbitalCalc.InclinationCosinus
	SinalMultiplication := anomalyEl.AnomalySinus * (math.Pow(orbitalCalc.InclinationCosinus, 2)*math.Cos(ascensionDiff) + math.Pow(orbitalCalc.InclinationSinus, 2))

	return CosinalMultplication + SinalMultiplication
}

func (orbitalCalc *OrbitalCalculations) IsOrbitAngleValid(angle float64) bool {
	return angle >= orbitalCalc.MinAscensionAngle && angle < orbitalCalc.MaxAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) getRealOrbitId(ascensionCalculated float64, orbitalAscension float64) (int, float64) {
	realId := int((ascensionCalculated - orbitalCalc.MinAscensionAngle) / orbitalCalc.AscensionStep)
	ascensionDiff := ascensionCalculated - orbitalAscension
	return realId, ascensionDiff
}

func (orbitalCalc *OrbitalCalculations) analyzeOrbitRange(LU float64, LD float64, rangeMin int, rangeMax int, inRangeOrbits *map[int]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements) {
	for i := rangeMin; i <= rangeMax; i++ {
		ascensionCalculated := math.Mod(orbitalCalc.AscensionStep*float64(i)+orbitalCalc.MinAscensionAngle+2*math.Pi, 2*math.Pi)
		if orbitalCalc.IsOrbitAngleValid(ascensionCalculated) {
			id, ascensionDiff := orbitalCalc.getRealOrbitId(ascensionCalculated, orbitalAscension)
			(*inRangeOrbits)[id] = OrbitCalc{
				CosinalCoefficient: orbitalCalc.calculateCosinalCoefficient(anomalyEl, ascensionDiff),
				SinalCoefficient:   orbitalCalc.calculateSinalCoefficient(anomalyEl, ascensionDiff),
				AscensionDiff:      ascensionDiff,
				OrbitalRange:       fmt.Sprintf("%f-%f", LU, LD),
			}
		}
	}
}

func (orbitalCalc *OrbitalCalculations) FindOrbitsInRange(lengthLimitRatio float64, currentAnomaly float64, anomalyEl AnomalyElements, orbitalAscension float64) map[int]OrbitCalc {
	inRangeOrbits := make(map[int]OrbitCalc)

	boundedAnomaly := math.Mod(currentAnomaly, 2*math.Pi)
	ascensionFromMin := orbitalAscension - orbitalCalc.MinAscensionAngle
	LD, LU := orbitalCalc.calculateLimits(lengthLimitRatio, anomalyEl.AnomalySinus)
	Phi := math.Atan(orbitalCalc.InclinationCosinus * anomalyEl.AnomalySinus / anomalyEl.AnomalyCosinus)

	firstRangeMin := int(math.Ceil((ascensionFromMin + Phi - LU) / orbitalCalc.AscensionStep))
	firstRangeMax := int(math.Floor((ascensionFromMin + Phi - LD) / orbitalCalc.AscensionStep))
	secondRangeMin := int(math.Ceil((ascensionFromMin + Phi + LD - math.Pi) / orbitalCalc.AscensionStep))
	secondRangeMax := int(math.Floor((ascensionFromMin + Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

	if boundedAnomaly > math.Pi/2.0 && boundedAnomaly <= 3.0*math.Pi/2.0 {
		firstRangeMin = int(math.Ceil((ascensionFromMin + Phi - LU + math.Pi) / orbitalCalc.AscensionStep))
		firstRangeMax = int(math.Floor((ascensionFromMin + Phi - LD + math.Pi) / orbitalCalc.AscensionStep))
		secondRangeMin = int(math.Ceil((ascensionFromMin + Phi + LD) / orbitalCalc.AscensionStep))
		secondRangeMax = int(math.Floor((ascensionFromMin + Phi + LU) / orbitalCalc.AscensionStep))
	}

	// Calculate First Range
	orbitalCalc.analyzeOrbitRange(LU, LD, firstRangeMin, firstRangeMax, &inRangeOrbits, orbitalAscension, anomalyEl)

	// Calculate Second Range
	orbitalCalc.analyzeOrbitRange(LU, LD, secondRangeMin, secondRangeMax, &inRangeOrbits, orbitalAscension, anomalyEl)

	return inRangeOrbits
}
