package helpers

/*
#cgo CFLAGS: -std=c99 -I.
#cgo LDFLAGS: -lm
#include "orbital_calculation.h"
#include <stdlib.h>
*/
import "C"

import (
	"math"
	"unsafe"
)

type OrbitalCalculations struct {
	InclinationSinus   float64
	InclinationCosinus float64
	NumberOfOrbits     int
	AscensionStep      float64 // in radians
	MinAscensionAngle  float64 // in radians
	MaxAscensionAngle  float64 // in radians
	UseGPU             bool
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
	GetMinAscensionAngle() float64
	GetMaxAscensionAngle() float64
	GetNumberOfOrbits() int
	FindOrbitsInRange(lengthLimitRatio float64, anomalyEl AnomalyElements, orbitalAscension float64, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc)
	findOrbitsInRange(lengthLimitRatio float64, anomalyEl AnomalyElements, orbitalAscension float64, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc)
	calculateCosinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64
	calculateSinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64
	convertOrbitIdToAscension(orbitId int) float64
	isOrbitAngleValid(angle float64) bool
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

func (orbitalCalc *OrbitalCalculations) GetMinAscensionAngle() float64 {
	return orbitalCalc.MinAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) GetMaxAscensionAngle() float64 {
	return orbitalCalc.MaxAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) GetNumberOfOrbits() int {
	return orbitalCalc.NumberOfOrbits
}

func (orbitalCalc *OrbitalCalculations) isOrbitAngleValid(angle float64) bool {
	return angle >= orbitalCalc.MinAscensionAngle && angle < orbitalCalc.MaxAscensionAngle
}

func (orbitalCalc *OrbitalCalculations) convertOrbitIdToAscension(orbitId int) float64 {
	return float64(orbitId)*orbitalCalc.AscensionStep + orbitalCalc.MinAscensionAngle
}

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

func (orbitalCalc *OrbitalCalculations) calculatePhi(anomalyEl AnomalyElements) float64 {
	return math.Atan2(orbitalCalc.InclinationCosinus*anomalyEl.AnomalySinus, anomalyEl.AnomalyCosinus)
}

func (orbitalCalc *OrbitalCalculations) calculateCosinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	cosinalMultplication := anomalyEl.AnomalyCosinus * math.Cos(ascensionDiff)
	sinalMultiplication := anomalyEl.AnomalySinus * math.Sin(ascensionDiff) * orbitalCalc.InclinationCosinus

	return sinalMultiplication - cosinalMultplication
}

func (orbitalCalc *OrbitalCalculations) calculateSinalCoefficient(anomalyEl AnomalyElements, ascensionDiff float64) float64 {
	cosinalMultplication := anomalyEl.AnomalyCosinus * math.Sin(ascensionDiff) * orbitalCalc.InclinationCosinus
	sinalMultiplication := anomalyEl.AnomalySinus * (math.Pow(orbitalCalc.InclinationCosinus, 2)*math.Cos(ascensionDiff) + math.Pow(orbitalCalc.InclinationSinus, 2))

	return cosinalMultplication + sinalMultiplication
}

func (orbitalCalc *OrbitalCalculations) analyzeOrbit(i int, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements) {
	ascensionCalculated := math.Mod(orbitalCalc.AscensionStep*float64(i)+orbitalCalc.MinAscensionAngle+4*math.Pi, 2*math.Pi)
	if orbitalCalc.isOrbitAngleValid(ascensionCalculated) {
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

func (orbitalCalc *OrbitalCalculations) analyzeOrbitRange(orbitRange Range, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc, orbitalAscension float64, anomalyEl AnomalyElements) {
	for i := orbitRange.Min; i <= orbitRange.Max; i++ {
		orbitalCalc.analyzeOrbit(i, inRangeIds, inRangeOrbits, orbitalAscension, anomalyEl)
	}
}

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

func (orbitalCalc *OrbitalCalculations) findOrbitsInRange(lengthLimitRatio float64, anomalyEl AnomalyElements,
	orbitalAscension float64, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc) {

	ascensionFromMin := orbitalAscension - orbitalCalc.MinAscensionAngle
	LD, LU := orbitalCalc.calculateLimits(lengthLimitRatio, anomalyEl.AnomalySinus)
	Phi := orbitalCalc.calculatePhi(anomalyEl)

	firstRange, secondRange := orbitalCalc.findRanges(LU, LD, Phi, ascensionFromMin)

	// Calculate First Range
	if firstRange.Min != 0 || firstRange.Max != -1 {
		orbitalCalc.analyzeOrbitRange(firstRange, inRangeIds, inRangeOrbits, orbitalAscension, anomalyEl)
	}

	// Calculate Second Range
	if secondRange.Min != 0 || secondRange.Max != -1 {
		orbitalCalc.analyzeOrbitRange(secondRange, inRangeIds, inRangeOrbits, orbitalAscension, anomalyEl)
	}
}

func (orbitalCalc *OrbitalCalculations) FindOrbitsInRange(lengthLimitRatio float64, anomalyEl AnomalyElements,
	orbitalAscension float64, inRangeIds *[]int, inRangeOrbits *[]OrbitCalc) {

	if orbitalCalc.UseGPU {
		var count C.int
		inRangeIdsC := (*C.int)(C.malloc(C.size_t(orbitalCalc.NumberOfOrbits) * C.size_t(unsafe.Sizeof(C.int(0)))))
		defer C.free(unsafe.Pointer(inRangeIdsC))
		inRangeOrbitsC := (*C.orbit_calc)(C.malloc(C.size_t(orbitalCalc.NumberOfOrbits) * C.size_t(unsafe.Sizeof(C.orbit_calc{}))))
		defer C.free(unsafe.Pointer(inRangeOrbitsC))
		anomalyElC := C.anomaly_elements{
			anomaly_sinus:   C.double(anomalyEl.AnomalySinus),
			anomaly_cosinus: C.double(anomalyEl.AnomalyCosinus),
		}
		calc := C.orbital_calculations{
			inclination_sinus:   C.double(orbitalCalc.InclinationSinus),
			inclination_cosinus: C.double(orbitalCalc.InclinationCosinus),
			ascension_step:      C.double(orbitalCalc.AscensionStep),
			min_ascension_angle: C.double(orbitalCalc.MinAscensionAngle),
			max_ascension_angle: C.double(orbitalCalc.MaxAscensionAngle),
			number_of_orbits:    C.int(orbitalCalc.NumberOfOrbits),
		}
		C.find_orbits_in_range(
			calc,
			C.double(lengthLimitRatio),
			anomalyElC,
			C.double(orbitalAscension),
			(*C.int)(unsafe.Pointer(inRangeIdsC)),
			(*C.orbit_calc)(unsafe.Pointer(inRangeOrbitsC)),
			&count,
		)

		for i := 0; i < int(count); i++ {
			*inRangeIds = append(*inRangeIds, int(((*[1 << 30]C.int)(unsafe.Pointer(inRangeIdsC)))[i]))
			*inRangeOrbits = append(*inRangeOrbits, OrbitCalc{
				CosinalCoefficient: float64(((*[1 << 30]C.orbit_calc)(unsafe.Pointer(inRangeOrbitsC)))[i].cosinal_coefficient),
				SinalCoefficient:   float64(((*[1 << 30]C.orbit_calc)(unsafe.Pointer(inRangeOrbitsC)))[i].sinal_coefficient),
				AscensionDiff:      float64(((*[1 << 30]C.orbit_calc)(unsafe.Pointer(inRangeOrbitsC)))[i].ascension_diff),
			})
		}
	} else {
		orbitalCalc.findOrbitsInRange(lengthLimitRatio, anomalyEl, orbitalAscension, inRangeIds, inRangeOrbits)
	}
}
