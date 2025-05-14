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
	FindCoordinatesOfTheAboveHeadPoint(gsName string, latitude float64, longitude float64) (float64, float64)
	FindSatellitesInRange(Id string, headPointAscension float64, headPointAnomalyEl AnomalyElements, timeStamp float64) map[string]float64
	UpdatePosition(prevAscension float64, timeStep float64) float64
	SetGroundStationSpecs(gsSpecs *GroundStationSpecs)
	GetCoveringGroundStations(timeStamp float64, anomaly float64, orbit IOrbit) map[string]float64
	GetAnomalyCalculations() IAnomalyCalculation
	adjustAngles(anomaly float64, deltaLongitude float64, adjustedLongitude float64) (float64, float64)
}

func (gsc *GroundStationCalculation) SetGroundStationSpecs(gsSpecs *GroundStationSpecs) {
	gsc.GroundStations = gsSpecs
}

func (gsc *GroundStationCalculation) GetAnomalyCalculations() IAnomalyCalculation {
	return gsc.AnomalyCalculations
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
	return prevAscension + gsc.EarthRotaionMotion*timeStep
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

func (gsc *GroundStationCalculation) FindSatellitesInRange(Id string, headPointAscension float64, headPointAnomalyEl AnomalyElements,
	timeStamp float64) map[string]float64 {

	satelliteDistances := gsc.AnomalyCalculations.FindSatellitesInRange(Id, gsc.ElevationLimitRatio,
		headPointAnomalyEl, headPointAscension, timeStamp)

	for id, distance := range satelliteDistances {
		updatedDistance := math.Sqrt(math.Pow(gsc.Altitude, 2.0) + gsc.EarthOrbitRatio*math.Pow(distance, 2.0))
		newDistanceObject := updatedDistance
		satelliteDistances[id] = newDistanceObject
	}

	return satelliteDistances
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

func (gsc *GroundStationCalculation) GetCoveringGroundStations(timeStamp float64, anomaly float64, orbit IOrbit) map[string]float64 {
	distances := make(map[string]float64)
	earthOrbitRatio := 1.0 - orbit.GetAltitude()/orbit.GetRadius()

	for gsName, gsSpec := range *gsc.GroundStations {
		gsAscension := gsSpec.HeadPointAscension + orbit.GetEarthRotaionMotion()*timeStamp
		distance := gsc.calculateGSDistance(gsSpec.HeadPointAnomalyEl, gsAscension, anomaly, orbit.GetAscension())

		if distance < gsc.GroundStationsDistanceLimit {
			updatedDistance := math.Sqrt(math.Pow(orbit.GetAltitude(), 2.0) + earthOrbitRatio*math.Pow(distance, 2.0))
			distances[gsName] = updatedDistance
		}
	}
	return distances
}
