package helpers

/*
#cgo CFLAGS: -std=c99 -I.
#cgo LDFLAGS: -lm
#include "anomaly_calculation.h"
#include "ground_station_calculation.h"
#include <stdlib.h>
*/
import "C"

import (
	"math"
	"unsafe"
)

type GroundStationSpec struct {
	Latitude           float64
	Longitude          float64
	HeadPointAscension float64
	HeadPointAnomalyEl AnomalyElements
}

type GroundStationSpecs map[string]GroundStationSpec

type GroundStationCalculation struct {
	AnomalyCalculations         IAnomalyCalculation
	ElevationLimitRatio         float64
	Altitude                    float64
	EarthOrbitRatio             float64
	EarthRotaionMotion          float64
	GroundStations              *GroundStationSpecs
	GroundStationCalculationC   *C.ground_station_calculation
	GroundStationsDistanceLimit float64
	UseGPU                      bool
}

type IGroundStationCalculation interface {
	GetAnomalyCalculations() IAnomalyCalculation
	FindCoordinatesOfTheAboveHeadPoint(gsName string, latitude float64, longitude float64) (float64, float64)
	FindSatellitesInRange(Id string, headPointAscension float64, headPointAnomalyEl AnomalyElements, timeStamp float64) map[string]float64
	UpdatePosition(prevAscension float64, timeStep float64) float64
	SetGroundStationSpecs(gsSpecs *GroundStationSpecs)
	GetCoveringGroundStations(timeStamp float64, anomaly float64, orbit IOrbit) map[string]float64
	coveringGroundStation(headPointAscension float64, headPointAnomalyEl AnomalyElements, anomaly float64, timeStamp float64,
		earthRotationMotion float64, earthOrbitRatio float64, ascension float64, altitude float64, distances *[]float64, gsInRange *[]string, gsName string)
	adjustAngles(anomaly float64, deltaLongitude float64, adjustedLongitude float64) (float64, float64)
	updateDistanceWithAltitude(i int, distances []float64, altitude float64, earthOrbitRatio float64)
}

func (gsc *GroundStationCalculation) SetGroundStationSpecs(gsSpecs *GroundStationSpecs) {
	gsc.GroundStations = gsSpecs
	gsc.initGroundStationCalcC()
}

func (gsc *GroundStationCalculation) GetAnomalyCalculations() IAnomalyCalculation {
	return gsc.AnomalyCalculations
}

func (gsc *GroundStationCalculation) initGroundStationCalcC() {
	groundCalcC := C.ground_station_calculation{
		elevation_limit_ratio:          C.double(gsc.ElevationLimitRatio),
		altitude:                       C.double(gsc.Altitude),
		earth_orbit_ratio:              C.double(gsc.EarthOrbitRatio),
		earth_rotation_motion:          C.double(gsc.EarthRotaionMotion),
		ground_stations_distance_limit: C.double(gsc.GroundStationsDistanceLimit),
		radius:                         C.double(gsc.AnomalyCalculations.GetRadius()),
		inclination_cosinus:            C.double(gsc.AnomalyCalculations.GetOrbitalCalculations().GetInclinationCosinus()),
		inclination_sinus:              C.double(gsc.AnomalyCalculations.GetOrbitalCalculations().GetInclinationSinus()),
		station_count:                  C.int(len(*gsc.GroundStations)),
	}
	i := 0
	for name, gsSpec := range *gsc.GroundStations {
		groundStationSpecC := C.ground_station_spec{
			latitude:             C.double(gsSpec.Latitude),
			longitude:            C.double(gsSpec.Longitude),
			head_point_ascension: C.double(gsSpec.HeadPointAscension),
			head_point_anomaly_el: C.anomaly_elements{
				anomaly_sinus:   C.double(gsSpec.HeadPointAnomalyEl.AnomalySinus),
				anomaly_cosinus: C.double(gsSpec.HeadPointAnomalyEl.AnomalyCosinus),
			},
		}
		groundCalcC.station_specs[i] = groundStationSpecC
		dest := (*[C.MAX_ID_LENGTH]byte)(unsafe.Pointer(&groundCalcC.station_names[i][0]))
		copy(dest[:], name)
		dest[len(name)] = 0
		i++
	}
	gsc.GroundStationCalculationC = &groundCalcC
}

func (gsc *GroundStationCalculation) adjustAngles(anomaly float64, deltaLongitude float64,
	adjustedLongitude float64) (float64, float64) {
	positiveAscension := adjustedLongitude + deltaLongitude
	negativeAscension := adjustedLongitude - deltaLongitude

	if anomaly < 0 {
		return math.Mod(positiveAscension, 2*math.Pi), 2*math.Pi + anomaly
	} else {
		return math.Mod(negativeAscension, 2*math.Pi), anomaly
	}
}

func (gsc *GroundStationCalculation) UpdatePosition(prevAscension float64, timeStep float64) float64 {
	if gsc.UseGPU {
		return float64(C.update_gs_position(C.double(prevAscension), C.double(timeStep), C.double(gsc.EarthRotaionMotion)))
	} else {
		return prevAscension + gsc.EarthRotaionMotion*timeStep
	}
}

func (gsc *GroundStationCalculation) FindCoordinatesOfTheAboveHeadPoint(gsName string, latitude float64, longitude float64) (float64, float64) {
	inclinationSinus := gsc.AnomalyCalculations.GetOrbitalCalculations().GetInclinationSinus()
	inclinationCosinus := gsc.AnomalyCalculations.GetOrbitalCalculations().GetInclinationCosinus()
	latitudeSinus := math.Sin(latitude)
	adjustedLongitude := math.Mod(longitude+2.0*math.Pi, 2.0*math.Pi)

	if math.Abs(latitudeSinus) >= inclinationSinus {
		if latitude < 0 {
			return math.Mod(adjustedLongitude+math.Pi/2.0, 2.0*math.Pi), 3.0 * math.Pi / 2.0
		} else {
			return math.Mod(adjustedLongitude+3.0*math.Pi/2.0, 2.0*math.Pi), math.Pi / 2.0
		}
	}
	anomaly := math.Asin(latitudeSinus / inclinationSinus)
	deltaLongitude := math.Abs(math.Atan(inclinationCosinus * math.Tan(anomaly)))

	return gsc.adjustAngles(anomaly, deltaLongitude, adjustedLongitude)
}

func (gsc *GroundStationCalculation) updateDistanceWithAltitude(i int, distances []float64, altitude float64, earthOrbitRatio float64) {
	distances[i] = math.Sqrt(math.Pow(altitude, 2.0) + earthOrbitRatio*math.Pow(distances[i], 2.0))
}

func (gsc *GroundStationCalculation) FindSatellitesInRange(Id string, headPointAscension float64, headPointAnomalyEl AnomalyElements,
	timeStamp float64) map[string]float64 {

	satelliteDistances := gsc.AnomalyCalculations.FindSatellitesInRange(Id, gsc.ElevationLimitRatio,
		headPointAnomalyEl, headPointAscension, timeStamp)

	satelliteIds, distances := unzip_satellite_ids_with_distances(satelliteDistances)

	if gsc.UseGPU && len(distances) > 0 {
		distancesC := make([]C.double, len(distances))
		for i, distance := range distances {
			distancesC[i] = C.double(distance)
		}
		C.update_distances_with_altitude((*C.double)(unsafe.Pointer(&distancesC[0])),
			C.int(len(distancesC)),
			C.double(gsc.Altitude),
			C.double(gsc.EarthOrbitRatio),
		)
		for i := range distances {
			distances[i] = float64(distancesC[i])
		}
	} else {
		for i := range distances {
			gsc.updateDistanceWithAltitude(i, distances, gsc.Altitude, gsc.EarthOrbitRatio)
		}
	}

	return zip_satellite_ids_with_distances(satelliteIds, distances)
}

func (gsc *GroundStationCalculation) calculateGSDistance(headPointAnomalyEl AnomalyElements, headPointAscension float64, anomaly float64, ascension float64) float64 {
	orbitalCalculations := gsc.AnomalyCalculations.GetOrbitalCalculations()
	ascensionDiff := headPointAscension - ascension

	orbitalCalc := OrbitCalc{
		CosinalCoefficient: orbitalCalculations.calculateCosinalCoefficient(headPointAnomalyEl, ascensionDiff),
		SinalCoefficient:   orbitalCalculations.calculateSinalCoefficient(headPointAnomalyEl, ascensionDiff),
		AscensionDiff:      ascensionDiff,
	}

	return gsc.AnomalyCalculations.CalculateDistance(orbitalCalc, anomaly)
}

func (gsc *GroundStationCalculation) coveringGroundStation(headPointAscension float64, headPointAnomalyEl AnomalyElements,
	anomaly float64, timeStamp float64, earthRotationMotion float64, earthOrbitRatio float64, ascension float64,
	altitude float64, distances *[]float64, gsInRange *[]string, gsName string) {
	gsAscension := headPointAscension + earthRotationMotion*timeStamp
	distance := gsc.calculateGSDistance(headPointAnomalyEl, gsAscension, anomaly, ascension)

	if distance < gsc.GroundStationsDistanceLimit {
		updatedDistance := math.Sqrt(math.Pow(altitude, 2.0) + earthOrbitRatio*math.Pow(distance, 2.0))
		*distances = append(*distances, updatedDistance)
		*gsInRange = append(*gsInRange, gsName)
	}
}

func (gsc *GroundStationCalculation) GetCoveringGroundStations(timeStamp float64, anomaly float64, orbit IOrbit) map[string]float64 {
	var distances []float64
	var gsInRange []string
	earthOrbitRatio := 1.0 - orbit.GetAltitude()/orbit.GetRadius()

	if gsc.UseGPU {
		var count C.int
		gsInRangeC := make([][int(C.MAX_ID_LENGTH)]C.char, int(C.MAX_GROUND_STATIONS))
		gsDistancesC := make([]C.double, int(C.MAX_GROUND_STATIONS))
		C.covering_ground_stations(gsc.GroundStationCalculationC, C.double(anomaly), C.double(timeStamp),
			C.double(earthOrbitRatio), C.double(orbit.GetAscension()), C.double(orbit.GetAltitude()),
			(*C.double)(unsafe.Pointer(&gsDistancesC[0])), (*[int(C.MAX_ID_LENGTH)]C.char)(unsafe.Pointer(&gsInRangeC[0])),
			&count)
		for i := 0; i < int(count); i++ {
			gsName := C.GoString(&gsInRangeC[i][0])
			gsInRange = append(gsInRange, gsName)
			distances = append(distances, float64(gsDistancesC[i]))
		}
	} else {
		for gsName, gsSpec := range *gsc.GroundStations {
			gsc.coveringGroundStation(gsSpec.HeadPointAscension, gsSpec.HeadPointAnomalyEl, anomaly, timeStamp,
				gsc.EarthRotaionMotion, earthOrbitRatio, orbit.GetAscension(), orbit.GetAltitude(), &distances, &gsInRange, gsName)
		}
	}

	return zip_gs_ids_with_distances(gsInRange, distances)
}

func zip_gs_ids_with_distances(gsIds []string, distances []float64) map[string]float64 {
	distancesMap := make(map[string]float64)
	for i, id := range gsIds {
		distancesMap[id] = distances[i]
	}
	return distancesMap
}

func unzip_satellite_ids_with_distances(distances map[string]float64) ([]string, []float64) {
	var satelliteIds []string
	var satelliteDistances []float64
	for id, distance := range distances {
		satelliteIds = append(satelliteIds, id)
		satelliteDistances = append(satelliteDistances, distance)
	}
	return satelliteIds, satelliteDistances
}
