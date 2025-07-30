package helpers

/*
#cgo CFLAGS: -std=c99 -I.
#cgo LDFLAGS: -lm
#include "anomaly_calculation.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"math"
	"unsafe"
)

type AnomalyElements struct {
	AnomalySinus   float64
	AnomalyCosinus float64
}

type IAnomalyCalculation interface {
	FindSatellitesInRange(Id string, lengthLimitRatio float64, anomalyEl AnomalyElements, orbitalAscension float64, timeStamp float64) map[string]float64
	UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements)
	CalculateDistanceBySatelliteId(firstSatelliteId int, firstSatelliteOrbitId int, secondSatelliteId int, secondSatelliteOrbitId int, timeStamp float64) float64
	CalculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64
	GetOrbitalCalculations() IOrbitalCalculations
	GetLengthLimitRatio() float64
	GetMaxDistance() float64
	GetRadius() float64
	findDistanceforSatelliteId(i int, baseId string, orbit int, timeStamp float64, orbitCalc OrbitCalc, initialPhaseShift float64, satelliteIds *[]string, satelliteDistances *[]float64)
	calculateSatelliteIdInRange(orbit int, orbitCalc OrbitCalc, baseId string, lengthLimitRatio float64, timeStamp float64, satelliteIds *[]string, satelliteDistances *[]float64)
	calculatePhase(satelliteId int, orbitId int) float64
	makeAnomalyCalculationsC(constellationName *C.char) C.anomaly_calculations
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
	UseGPU                     bool
}

func (anomalyCalc *AnomalyCalculations) GetOrbitalCalculations() IOrbitalCalculations {
	return anomalyCalc.OrbitalCalculations
}

func (anomalyCalc *AnomalyCalculations) GetLengthLimitRatio() float64 {
	return anomalyCalc.LengthLimitRatio
}

func (anomalyCalc *AnomalyCalculations) GetRadius() float64 {
	return anomalyCalc.Radius
}

func (anomalyCalc *AnomalyCalculations) GetMaxDistance() float64 {
	return anomalyCalc.MaxDistance
}

func (anomalyCalc *AnomalyCalculations) CalculateDistance(orbitCalc OrbitCalc, otherSatelliteAnomaly float64) float64 {
	if anomalyCalc.UseGPU {
		orbitCalcC := C.orbit_calc{
			cosinal_coefficient: C.double(orbitCalc.CosinalCoefficient),
			sinal_coefficient:   C.double(orbitCalc.SinalCoefficient),
			ascension_diff:      C.double(orbitCalc.AscensionDiff),
		}
		return float64(C.calculate_distance(C.double(anomalyCalc.Radius), orbitCalcC, C.double(otherSatelliteAnomaly)))
	} else {
		distance_squared_factor := 2 * (orbitCalc.CosinalCoefficient*math.Cos(otherSatelliteAnomaly) -
			orbitCalc.SinalCoefficient*math.Sin(otherSatelliteAnomaly) + 1.0)

		if distance_squared_factor <= 0.0 {
			return 0.0
		}

		return anomalyCalc.Radius * math.Sqrt(distance_squared_factor)
	}
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

func convertOrbitsToC(orbitIds []int, orbitCalcs []OrbitCalc) (*C.int, *C.orbit_calc, C.int) {
	orbit_count := len(orbitIds)
	orbitIdsC := make([]C.int, orbit_count)
	orbitCalcsC := make([]C.orbit_calc, orbit_count)
	for i := 0; i < orbit_count; i++ {
		orbitIdsC[i] = C.int(orbitIds[i])
		orbitCalcsC[i] = C.orbit_calc{
			cosinal_coefficient: C.double(orbitCalcs[i].CosinalCoefficient),
			sinal_coefficient:   C.double(orbitCalcs[i].SinalCoefficient),
			ascension_diff:      C.double(orbitCalcs[i].AscensionDiff),
		}
	}
	return (*C.int)(unsafe.Pointer(&orbitIdsC[0])), (*C.orbit_calc)(unsafe.Pointer(&orbitCalcsC[0])), C.int(orbit_count)
}

func (anomalyCalc *AnomalyCalculations) FindSatellitesInRange(Id string, lengthLimitRatio float64, anomalyEl AnomalyElements,
	orbitalAscension float64, timeStamp float64) map[string]float64 {
	var satelliteDistances []float64
	var satelliteIds []string
	var inRangeIds []int
	var inRangeOrbits []OrbitCalc
	anomalyCalc.OrbitalCalculations.FindOrbitsInRange(lengthLimitRatio, anomalyEl, orbitalAscension, &inRangeIds, &inRangeOrbits)
	if anomalyCalc.UseGPU {
		numberOfSatellites := anomalyCalc.NumberOfSatellitesPerOrbit * anomalyCalc.OrbitalCalculations.GetNumberOfOrbits()
		var count C.int
		satelliteIdsC := make([][int(C.MAX_ID_LENGTH)]C.char, numberOfSatellites)
		satelliteDistancesC := make([]C.double, numberOfSatellites)
		baseIdC := C.CString(Id)
		defer C.free(unsafe.Pointer(baseIdC))
		constellationName := C.CString(anomalyCalc.ConsellationName)
		defer C.free(unsafe.Pointer(constellationName))
		anomalyCalcC := anomalyCalc.makeAnomalyCalculationsC(constellationName)
		inRangeIdsC, inRangeOrbitsC, orbitCount := convertOrbitsToC(inRangeIds, inRangeOrbits)
		C.find_satellites_in_range(&anomalyCalcC, inRangeIdsC, inRangeOrbitsC, orbitCount, baseIdC, C.double(lengthLimitRatio),
			C.double(timeStamp), (*[int(C.MAX_ID_LENGTH)]C.char)(unsafe.Pointer(&satelliteIdsC[0])),
			(*C.double)(unsafe.Pointer(&satelliteDistancesC[0])), &count)
		for i := 0; i < int(count); i++ {
			satelliteId := C.GoString(&satelliteIdsC[i][0])
			satelliteIds = append(satelliteIds, satelliteId)
			satelliteDistances = append(satelliteDistances, float64(satelliteDistancesC[i]))
		}
	} else {
		for i := 0; i < len(inRangeIds); i++ {
			anomalyCalc.calculateSatelliteIdInRange(inRangeIds[i], inRangeOrbits[i], Id, lengthLimitRatio, timeStamp, &satelliteIds, &satelliteDistances)
		}
	}
	return zip_satellite_ids_with_distances(satelliteIds, satelliteDistances)
}

func (anomalyCalc *AnomalyCalculations) UpdatePosition(prevAnomaly float64, timeStep float64) (float64, AnomalyElements) {
	if anomalyCalc.UseGPU {
		var newOrbitalAnomaly C.double
		newAnomalyElements := C.anomaly_elements{}
		C.update_sat_position(C.double(anomalyCalc.MeanMotion), C.double(prevAnomaly), C.double(timeStep), &newOrbitalAnomaly, &newAnomalyElements)
		return float64(newOrbitalAnomaly), AnomalyElements{
			AnomalySinus:   float64(newAnomalyElements.anomaly_sinus),
			AnomalyCosinus: float64(newAnomalyElements.anomaly_cosinus),
		}
	} else {
		newOrbitalAnomaly := prevAnomaly + anomalyCalc.MeanMotion*timeStep
		newAnomalyElements := AnomalyElements{
			AnomalySinus:   math.Sin(newOrbitalAnomaly),
			AnomalyCosinus: math.Cos(newOrbitalAnomaly),
		}
		return newOrbitalAnomaly, newAnomalyElements
	}
}

func (anomalyCalc *AnomalyCalculations) calculatePhase(satelliteId int, orbitId int) float64 {
	phase := float64(satelliteId) * anomalyCalc.AnomalyStep
	if anomalyCalc.PhaseDiffEnabled && orbitId%2 == 1 {
		phase += anomalyCalc.AnomalyStep / 2.0
	}

	return phase
}

// Timestamp should be in seconds
func (anomalyCalc *AnomalyCalculations) CalculateDistanceBySatelliteId(firstSatelliteId int, firstSatelliteOrbitId int,
	secondSatelliteId int, secondSatelliteOrbitId int, timeStamp float64) float64 {

	if anomalyCalc.UseGPU {
		constellationName := C.CString(anomalyCalc.ConsellationName)
		defer C.free(unsafe.Pointer(constellationName))
		anomalyCalcC := anomalyCalc.makeAnomalyCalculationsC(constellationName)
		return float64(C.calculate_distance_by_satellite_id(&anomalyCalcC, C.int(firstSatelliteId),
			C.int(firstSatelliteOrbitId), C.int(secondSatelliteId), C.int(secondSatelliteOrbitId), C.double(timeStamp)))
	} else {
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
}

func (anomalyCalc *AnomalyCalculations) makeAnomalyCalculationsC(constellationName *C.char) C.anomaly_calculations {
	var phaseDiff C.int
	if anomalyCalc.PhaseDiffEnabled {
		phaseDiff = C.int(1)
	} else {
		phaseDiff = C.int(0)
	}
	return C.anomaly_calculations{
		constellation_name:             constellationName,
		number_of_satellites_per_orbit: C.int(anomalyCalc.NumberOfSatellitesPerOrbit),
		anomaly_step:                   C.double(anomalyCalc.AnomalyStep),
		mean_motion:                    C.double(anomalyCalc.MeanMotion),
		radius:                         C.double(anomalyCalc.Radius),
		orbital_calc: C.orbital_calculations{
			inclination_sinus:   C.double(anomalyCalc.OrbitalCalculations.GetInclinationSinus()),
			inclination_cosinus: C.double(anomalyCalc.OrbitalCalculations.GetInclinationCosinus()),
			ascension_step:      C.double(anomalyCalc.OrbitalCalculations.GetAscensionStep()),
			min_ascension_angle: C.double(anomalyCalc.OrbitalCalculations.GetMinAscensionAngle()),
			max_ascension_angle: C.double(anomalyCalc.OrbitalCalculations.GetMaxAscensionAngle()),
			number_of_orbits:    C.int(anomalyCalc.OrbitalCalculations.GetNumberOfOrbits()),
		},
		phase_diff_enabled: phaseDiff,
	}
}

func zip_satellite_ids_with_distances(inRangeIds []string, distances []float64) map[string]float64 {
	distancesMap := make(map[string]float64)
	for i, id := range inRangeIds {
		distancesMap[id] = distances[i]
	}
	return distancesMap
}
