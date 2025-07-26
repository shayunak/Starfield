package helpers

import (
	"math"
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
	GroundStations              *GroundStationSpecs // groundStations
	GroundStationsDistanceLimit float64
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
	update_distance_with_altitude(i int, distances []float64, altitude float64, earthOrbitRatio float64)
}

func (gsc *GroundStationCalculation) SetGroundStationSpecs(gsSpecs *GroundStationSpecs) {
	gsc.GroundStations = gsSpecs
}

func (gsc *GroundStationCalculation) GetAnomalyCalculations() IAnomalyCalculation {
	return gsc.AnomalyCalculations
}

// Cuda Compatible
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
	return prevAscension + gsc.EarthRotaionMotion*timeStep
}

// cuda Compatible
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

// Cuda Compatible
func (gsc *GroundStationCalculation) update_distance_with_altitude(i int, distances []float64, altitude float64, earthOrbitRatio float64) {
	distances[i] = math.Sqrt(math.Pow(altitude, 2.0) + earthOrbitRatio*math.Pow(distances[i], 2.0))
}

func (gsc *GroundStationCalculation) FindSatellitesInRange(Id string, headPointAscension float64, headPointAnomalyEl AnomalyElements,
	timeStamp float64) map[string]float64 {

	satelliteDistances := gsc.AnomalyCalculations.FindSatellitesInRange(Id, gsc.ElevationLimitRatio,
		headPointAnomalyEl, headPointAscension, timeStamp)

	satelliteIds, distances := unzip_satellite_ids_with_distances(satelliteDistances)

	for i := range distances {
		gsc.update_distance_with_altitude(i, distances, gsc.Altitude, gsc.EarthOrbitRatio)
	}

	return zip_satellite_ids_with_distances(satelliteIds, distances)
}

// Cuda Compatible
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

// Cuda Compatible
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

	for gsName, gsSpec := range *gsc.GroundStations {
		gsc.coveringGroundStation(gsSpec.HeadPointAscension, gsSpec.HeadPointAnomalyEl, anomaly, timeStamp,
			gsc.EarthRotaionMotion, earthOrbitRatio, orbit.GetAscension(), orbit.GetAltitude(), &distances, &gsInRange, gsName)
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
