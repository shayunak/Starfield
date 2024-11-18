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
	AnomalyCalculations IAnomalyCalculation
	ElevationLimitRatio float64
	Altitude            float64
	EarthOrbitRatio     float64
	EarthRotaionMotion  float64
}

type IGroundStationCalculation interface {
	FindCoordinatesOfTheAboveHeadPoint(latitude float64, longitude float64) (float64, float64)
	FindSatellitesInRange(headPointAnomaly float64, headPointAscension float64, headPointAnomalyEl AnomalyElements, timeStamp float64) map[string]DistanceObject
	adjustAngles(anomaly float64, deltaLongitude float64, adjustedLongitude float64) (float64, float64)
	UpdatePosition(prevAscension float64, timeStep float64) float64
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

func (gsc *GroundStationCalculation) FindCoordinatesOfTheAboveHeadPoint(latitude float64, longitude float64) (float64, float64) {
	inclinationSinus := gsc.AnomalyCalculations.GetOrbitalCalculations().GetInclinationSinus()
	inclinationCosinus := gsc.AnomalyCalculations.GetOrbitalCalculations().GetInclinationCosinus()
	latitudeSinus := math.Sin(latitude)
	adjustedLongitude := math.Mod(longitude+2*math.Pi, 2*math.Pi)

	if math.Abs(latitudeSinus) >= inclinationSinus {
		if latitude < 0 {
			return math.Mod(longitude+3.0*math.Pi/2.0, 2*math.Pi), 3.0 * math.Pi / 2.0
		} else {
			return math.Mod(longitude+math.Pi/2.0, 2*math.Pi), math.Pi / 2.0
		}
	}
	anomaly := math.Asin(latitudeSinus / inclinationSinus)
	deltaLongitude := math.Abs(math.Atan(inclinationCosinus * math.Tan(anomaly)))

	return gsc.adjustAngles(anomaly, deltaLongitude, adjustedLongitude)
}

func (gsc *GroundStationCalculation) FindSatellitesInRange(headPointAnomaly float64, headPointAscension float64,
	headPointAnomalyEl AnomalyElements, timeStamp float64) map[string]DistanceObject {

	satelliteDistances := gsc.AnomalyCalculations.FindSatellitesInRange(gsc.ElevationLimitRatio, headPointAnomaly,
		headPointAnomalyEl, headPointAscension, timeStamp)

	for id, distanceObject := range satelliteDistances {
		updatedDistance := math.Sqrt(math.Pow(gsc.Altitude, 2.0) + gsc.EarthOrbitRatio*math.Pow(distanceObject.Distance, 2.0))
		newDistanceObject := DistanceObject{
			Distance:      updatedDistance,
			Anomaly:       distanceObject.Anomaly,
			AscensionDiff: distanceObject.AscensionDiff,
			A:             distanceObject.A,
			B:             distanceObject.B,
		}
		satelliteDistances[id] = newDistanceObject
	}

	return satelliteDistances
}
