package helpers

import (
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
}

type Range struct {
	Min int
	Max int
}

type IOrbitalCalculations interface {
	GetInclinationSinus() float64
	GetInclinationCosinus() float64
	GetAscensionStep() float64
	FindOrbitsInRange(lengthLimitRatio float64, anomalyEl AnomalyElements, orbitalAscension float64) map[int]OrbitCalc
	calculateCosinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64
	calculateSinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64
	ConvertOrbitIdToAscension(orbitId int) float64
	IsOrbitAngleValid(angle float64) bool
	analyzeOrbit(i int, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements)
	analyzeOrbitRange(orbitRange Range, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements)
	calculateLimits(lengthLimitRatio float64, anomalySinus float64) (float64, float64)
	calculatePhi(anomalyEl AnomalyElements) float64
	findRanges(LU float64, LD float64, Phi float64, ascensionFromMin float64) (Range, Range)
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

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) IsOrbitAngleValid(angle float64) bool {
	return angle >= orbitalCalc.MinAscensionAngle && angle <= orbitalCalc.MaxAscensionAngle
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) ConvertOrbitIdToAscension(orbitId int) float64 {
	return float64(orbitId)*orbitalCalc.AscensionStep + orbitalCalc.MinAscensionAngle
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) calculateLimits(lengthLimitRatio float64, anomalySinus float64) (float64, float64) {

	ISLLengthLimit := math.Sqrt(1 - math.Pow(lengthLimitRatio, 2.0))
	baseTrig := orbitalCalc.InclinationSinus * orbitalCalc.InclinationCosinus * anomalySinus
	denominator := orbitalCalc.InclinationSinus * math.Sqrt(1.0-math.Pow(anomalySinus*orbitalCalc.InclinationSinus, 2.0))
	lowerLimit := (baseTrig - ISLLengthLimit) / denominator
	upperLimit := (baseTrig + ISLLengthLimit) / denominator
	LU := math.Pi / 2.0
	LD := -math.Pi / 2.0

	if lowerLimit > -1.0 && lowerLimit < 1.0 {
		LD = math.Asin(lowerLimit)
	} else if lowerLimit > 1.0 {
		LD = math.Pi / 2.0
	}

	if upperLimit > -1.0 && upperLimit < 1.0 {
		LU = math.Asin(upperLimit)
	} else if upperLimit < -1.0 {
		LU = -math.Pi / 2.0
	}

	return LD, LU
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) calculatePhi(anomalyEl AnomalyElements) float64 {
	return math.Atan2(orbitalCalc.InclinationCosinus*anomalyEl.AnomalySinus, anomalyEl.AnomalyCosinus)
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) calculateCosinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	cosinalMultplication := anomalyEl.AnomalyCosinus * math.Cos(ascensionDiff)
	sinalMultiplication := anomalyEl.AnomalySinus * math.Sin(ascensionDiff) * orbitalCalc.InclinationCosinus

	return sinalMultiplication - cosinalMultplication
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) calculateSinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	cosinalMultplication := anomalyEl.AnomalyCosinus * math.Sin(ascensionDiff) * orbitalCalc.InclinationCosinus
	sinalMultiplication := anomalyEl.AnomalySinus * (math.Pow(orbitalCalc.InclinationCosinus, 2)*math.Cos(ascensionDiff) + math.Pow(orbitalCalc.InclinationSinus, 2))

	return cosinalMultplication + sinalMultiplication
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) analyzeOrbit(i int, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements) {
	ascensionCalculated := math.Mod(orbitalCalc.AscensionStep*float64(i)+orbitalCalc.MinAscensionAngle+2*math.Pi, 2*math.Pi)
	if orbitalCalc.IsOrbitAngleValid(ascensionCalculated) {
		ascensionDiff := orbitalAscension - ascensionCalculated
		realId := int(math.Round((ascensionCalculated - orbitalCalc.MinAscensionAngle) / orbitalCalc.AscensionStep))
		*inRangeIds = append(*inRangeIds, realId)
		*inRangeOrbits = append(*inRangeOrbits, OrbitCalc{
			CosinalCoefficient: orbitalCalc.calculateCosinalCoefficient(anomalyEl, ascensionDiff),
			SinalCoefficient:   orbitalCalc.calculateSinalCoefficient(anomalyEl, ascensionDiff),
			AscensionDiff:      ascensionDiff,
		})
	}
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) analyzeOrbitRange(orbitRange Range, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements) {
	for i := orbitRange.Min; i <= orbitRange.Max; i++ {
		orbitalCalc.analyzeOrbit(i, inRangeIds, inRangeOrbits, orbitalAscension, anomalyEl)
	}
}

// Cuda Compatible
func (orbitalCalc *OrbitalCalculations) findRanges(LU float64, LD float64, Phi float64, ascensionFromMin float64) (Range, Range) {
	var firstRange Range
	var secondRange Range

	if LD > -math.Pi/2.0 && LU < math.Pi/2.0 {
		firstRangeMin := int(math.Ceil((ascensionFromMin + Phi - LU) / orbitalCalc.AscensionStep))
		firstRangeMax := int(math.Floor((ascensionFromMin + Phi - LD) / orbitalCalc.AscensionStep))
		secondRangeMin := int(math.Ceil((ascensionFromMin + Phi + LD - math.Pi) / orbitalCalc.AscensionStep))
		secondRangeMax := int(math.Floor((ascensionFromMin + Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

		firstRange = Range{Min: firstRangeMin, Max: firstRangeMax}
		secondRange = Range{Min: secondRangeMin, Max: secondRangeMax}
	} else if LD <= -math.Pi/2.0 && LU < math.Pi/2.0 && LU > -math.Pi/2.0 {
		firstRangeMin := int(math.Ceil((ascensionFromMin + Phi - LU - 2*math.Pi) / orbitalCalc.AscensionStep))
		firstRangeMax := int(math.Floor((ascensionFromMin + Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

		firstRange = Range{Min: firstRangeMin, Max: firstRangeMax}
		secondRange = Range{Min: 0, Max: -1}
	} else if LD > -math.Pi/2.0 && LD < math.Pi/2.0 && LU >= math.Pi/2.0 {
		firstRangeMin := int(math.Ceil((ascensionFromMin + Phi + LD - math.Pi) / orbitalCalc.AscensionStep))
		firstRangeMax := int(math.Floor((ascensionFromMin + Phi - LD) / orbitalCalc.AscensionStep))
		firstRange = Range{Min: firstRangeMin, Max: firstRangeMax}
		secondRange = Range{Min: 0, Max: -1}
	} else if LU >= math.Pi/2.0 && LD <= -math.Pi/2.0 {
		firstRange = Range{Min: 0, Max: orbitalCalc.NumberOfOrbits - 1}
		secondRange = Range{Min: 0, Max: -1}
	} else {
		firstRange = Range{Min: 0, Max: -1}
		secondRange = Range{Min: 0, Max: -1}
	}

	return firstRange, secondRange
}

func (orbitalCalc *OrbitalCalculations) FindOrbitsInRange(lengthLimitRatio float64, anomalyEl AnomalyElements, orbitalAscension float64) map[int]OrbitCalc {
	var inRangeIds []int
	var inRangeOrbits []OrbitCalc

	ascensionFromMin := orbitalAscension - orbitalCalc.MinAscensionAngle
	LD, LU := orbitalCalc.calculateLimits(lengthLimitRatio, anomalyEl.AnomalySinus)
	Phi := orbitalCalc.calculatePhi(anomalyEl)

	firstRange, secondRange := orbitalCalc.findRanges(LU, LD, Phi, ascensionFromMin)

	// Calculate First Range
	if firstRange.Min != 0 || firstRange.Max != -1 {
		orbitalCalc.analyzeOrbitRange(firstRange, &inRangeIds, &inRangeOrbits, orbitalAscension, anomalyEl)
	}

	// Calculate Second Range
	if secondRange.Min != 0 || secondRange.Max != -1 {
		orbitalCalc.analyzeOrbitRange(secondRange, &inRangeIds, &inRangeOrbits, orbitalAscension, anomalyEl)
	}

	return zip_orbit_ids_with_orbit_calculations(inRangeIds, inRangeOrbits)
}

func zip_orbit_ids_with_orbit_calculations(inRangeIds []int, inRangeOrbits []OrbitCalc) map[int]OrbitCalc {
	inRangeOrbitsMap := make(map[int]OrbitCalc)
	for i, id := range inRangeIds {
		inRangeOrbitsMap[id] = inRangeOrbits[i]
	}
	return inRangeOrbitsMap
}
