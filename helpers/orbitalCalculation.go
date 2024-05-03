package helpers

import "math"

type OrbitalCalculations struct {
	InclinationSinus   float64
	InclinationCosinus float64
	LengthLimitRatio   float64
	NumberOfOrbits     int
	AscensionStep      float64 // in radians
}

type OrbitCalc struct {
	CosinalCoefficient float64
	SinalCoefficient   float64
}

type IOrbitalCalculations interface {
	FindOrbitsInRange(anomalyEl AnomalyElements, orbit_id int) map[int]OrbitCalc
}

func calculateLimits(lengthLimitRatio float64, inclinationSinus float64, inclinationCosinus float64,
	anomalySinus float64) (float64, float64) {

	ISLLengthLimit := math.Sqrt(1 - math.Pow(lengthLimitRatio, 2))
	baseTrig := inclinationSinus * inclinationCosinus * anomalySinus
	denominator := inclinationSinus * math.Sqrt(1-math.Pow(anomalySinus*inclinationSinus, 2))

	return math.Asin((baseTrig - ISLLengthLimit) / denominator), math.Asin((baseTrig + ISLLengthLimit) / denominator)
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

func (orbitalCalc *OrbitalCalculations) FindOrbitsInRange(anomalyEl AnomalyElements, orbit_id int) map[int]OrbitCalc {
	inRangeOrbits := make(map[int]OrbitCalc)

	LD, LU := calculateLimits(orbitalCalc.LengthLimitRatio, orbitalCalc.InclinationSinus,
		orbitalCalc.InclinationCosinus, anomalyEl.AnomalySinus)
	Phi := math.Atan(orbitalCalc.InclinationCosinus * anomalyEl.AnomalySinus / anomalyEl.AnomalyCosinus)
	firstRangeMin := int(math.Ceil((Phi - LU) / orbitalCalc.AscensionStep))
	firstRangeMax := int(math.Floor((Phi - LD) / orbitalCalc.AscensionStep))
	secondRangeMin := int(math.Ceil((Phi + LD - math.Pi) / orbitalCalc.AscensionStep))
	secondRangeMax := int(math.Floor((Phi + LU - math.Pi) / orbitalCalc.AscensionStep))

	// Calculate First Range
	for i := firstRangeMin; i <= firstRangeMax; i++ {
		id := (orbit_id + i + orbitalCalc.NumberOfOrbits) % orbitalCalc.NumberOfOrbits
		ascensionDiff := -1 * orbitalCalc.AscensionStep * float64(i)
		inRangeOrbits[id] = OrbitCalc{
			CosinalCoefficient: calculateCosinalCoefficient(orbitalCalc.InclinationCosinus, anomalyEl, ascensionDiff),
			SinalCoefficient:   calculateSinalCoefficient(orbitalCalc.InclinationCosinus, orbitalCalc.InclinationSinus, anomalyEl, ascensionDiff),
		}
	}

	// Calculate Second Range
	for i := secondRangeMin; i <= secondRangeMax; i++ {
		id := (orbit_id + i + orbitalCalc.NumberOfOrbits) % orbitalCalc.NumberOfOrbits
		ascensionDiff := -1 * orbitalCalc.AscensionStep * float64(i)
		inRangeOrbits[id] = OrbitCalc{
			CosinalCoefficient: calculateCosinalCoefficient(orbitalCalc.InclinationCosinus, anomalyEl, ascensionDiff),
			SinalCoefficient:   calculateSinalCoefficient(orbitalCalc.InclinationCosinus, orbitalCalc.InclinationSinus, anomalyEl, ascensionDiff),
		}
	}

	return inRangeOrbits
}
