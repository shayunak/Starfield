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

type Range struct {
	Min int
	Max int
}

type IOrbitalCalculations interface {
	GetInclinationSinus() float64
	GetInclinationCosinus() float64
	GetAscensionStep() float64
	FindOrbitsInRange(lengthLimitRatio float64, currentAnomaly float64, anomalyEl AnomalyElements, orbitalAscension float64) map[int]OrbitCalc
	IsOrbitAngleValid(angle float64) bool
	analyzeOrbitRange(LU float64, LD float64, currentAnomaly float64, rangeMin int, rangeMax int, inRangeOrbits *map[int]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements)
	calculateLimits(lengthLimitRatio float64, anomalySinus float64) (float64, float64)
	findRanges(LU float64, LD float64, Phi float64, ascensionFromMin float64, boundedAnomaly float64) (Range, Range)
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

	ISLLengthLimit := math.Sqrt(1 - math.Pow(lengthLimitRatio, 2.0))
	baseTrig := orbitalCalc.InclinationSinus * orbitalCalc.InclinationCosinus * anomalySinus
	denominator := orbitalCalc.InclinationSinus * math.Sqrt(1.0-math.Pow(anomalySinus*orbitalCalc.InclinationSinus, 2.0))
	lowerLimit := (baseTrig - ISLLengthLimit) / denominator
	upperLimit := (baseTrig + ISLLengthLimit) / denominator
	LU := 1.0
	LD := -1.0

	if lowerLimit > -1.0 && lowerLimit < 1.0 {
		LD = math.Asin(lowerLimit)
	} else if lowerLimit > 1.0 {
		LD = 1.0
	}

	if upperLimit > -1.0 && upperLimit < 1.0 {
		LU = math.Asin(upperLimit)
	} else if upperLimit < -1.0 {
		LU = -1.0
	}

	return LD, LU
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
	return angle >= orbitalCalc.MinAscensionAngle && angle <= orbitalCalc.MaxAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) analyzeOrbitRange(LU float64, LD float64, currentAnomaly float64, rangeMin int, rangeMax int, inRangeOrbits *map[int]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements) {
	for i := rangeMin; i <= rangeMax; i++ {
		ascensionCalculated := math.Mod(orbitalCalc.AscensionStep*float64(i)+orbitalCalc.MinAscensionAngle+2*math.Pi, 2*math.Pi)
		if orbitalCalc.IsOrbitAngleValid(ascensionCalculated) {
			ascensionDiff := ascensionCalculated - orbitalAscension
			realId := int(math.Round((ascensionCalculated - orbitalCalc.MinAscensionAngle) / orbitalCalc.AscensionStep))
			(*inRangeOrbits)[realId] = OrbitCalc{
				CosinalCoefficient: orbitalCalc.calculateCosinalCoefficient(anomalyEl, ascensionDiff),
				SinalCoefficient:   orbitalCalc.calculateSinalCoefficient(anomalyEl, ascensionDiff),
				AscensionDiff:      ascensionDiff,
				OrbitalRange:       fmt.Sprintf("%f-%f-%f,%f,%f,%d,%d,%d,%f", ascensionDiff, LU, LD, orbitalAscension, currentAnomaly, rangeMax, rangeMin, realId, ascensionCalculated),
			}
		}
	}
}

func (orbitalCalc *OrbitalCalculations) findRanges(LU float64, LD float64, Phi float64, ascensionFromMin float64, boundedAnomaly float64) (Range, Range) {
	var firstRange Range
	var secondRange Range

	if LD > -1.0 && LU < 1.0 {
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
		firstRange = Range{Min: firstRangeMin, Max: firstRangeMax}
		secondRange = Range{Min: secondRangeMin, Max: secondRangeMax}
	} else if LD <= -1.0 && LU < 1.0 && LU > -1.0 {
		firstRangeMin := int(math.Ceil((ascensionFromMin + Phi - LU - 2*math.Pi) / orbitalCalc.AscensionStep))
		firstRangeMax := int(math.Floor((ascensionFromMin + Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

		if boundedAnomaly > math.Pi/2.0 && boundedAnomaly <= 3.0*math.Pi/2.0 {
			firstRangeMin = int(math.Ceil((ascensionFromMin + Phi - LU - math.Pi) / orbitalCalc.AscensionStep))
			firstRangeMax = int(math.Floor((ascensionFromMin + Phi + LU) / orbitalCalc.AscensionStep))
		}
		firstRange = Range{Min: firstRangeMin, Max: firstRangeMax}
		secondRange = Range{Min: 0, Max: -1}
	} else if LD > -1.0 && LD < 1.0 && LU >= 1.0 {
		firstRangeMin := int(math.Ceil((ascensionFromMin + Phi - LU - 2*math.Pi) / orbitalCalc.AscensionStep))
		firstRangeMax := int(math.Floor((ascensionFromMin + Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

		if boundedAnomaly > math.Pi/2.0 && boundedAnomaly <= 3.0*math.Pi/2.0 {
			firstRangeMin = int(math.Ceil((ascensionFromMin + Phi + LD - math.Pi) / orbitalCalc.AscensionStep))
			firstRangeMax = int(math.Floor((ascensionFromMin + Phi - LD) / orbitalCalc.AscensionStep))
		}
		firstRange = Range{Min: firstRangeMin, Max: firstRangeMax}
		secondRange = Range{Min: 0, Max: -1}
	} else if LU >= 1.0 && LD <= -1.0 {
		firstRange = Range{Min: 0, Max: orbitalCalc.NumberOfOrbits - 1}
		secondRange = Range{Min: 0, Max: -1}
	} else {
		firstRange = Range{Min: 0, Max: -1}
		secondRange = Range{Min: 0, Max: -1}
	}

	return firstRange, secondRange
}

func (orbitalCalc *OrbitalCalculations) FindOrbitsInRange(lengthLimitRatio float64, currentAnomaly float64, anomalyEl AnomalyElements, orbitalAscension float64) map[int]OrbitCalc {
	inRangeOrbits := make(map[int]OrbitCalc)

	boundedAnomaly := math.Mod(currentAnomaly+2*math.Pi, 2*math.Pi)
	ascensionFromMin := orbitalAscension - orbitalCalc.MinAscensionAngle
	LD, LU := orbitalCalc.calculateLimits(lengthLimitRatio, anomalyEl.AnomalySinus)
	Phi := math.Atan(orbitalCalc.InclinationCosinus * anomalyEl.AnomalySinus / anomalyEl.AnomalyCosinus)

	firstRange, secondRange := orbitalCalc.findRanges(LU, LD, Phi, ascensionFromMin, boundedAnomaly)

	// Calculate First Range
	if firstRange.Min != 0 || firstRange.Max != -1 {
		orbitalCalc.analyzeOrbitRange(LU, LD, currentAnomaly, firstRange.Min, firstRange.Max, &inRangeOrbits, orbitalAscension, anomalyEl)
	}

	// Calculate Second Range
	if secondRange.Min != 0 || secondRange.Max != -1 {
		orbitalCalc.analyzeOrbitRange(LU, LD, currentAnomaly, secondRange.Min, secondRange.Max, &inRangeOrbits, orbitalAscension, anomalyEl)
	}

	return inRangeOrbits
}
