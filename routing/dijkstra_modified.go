package routing

import (
	"math"
	"slices"

	"Starfield/helpers"
)

func DijkstraModifiedOnGridPlus(nextBestHop string, timeStamp float64, interfaces []string, anomalyCalculation helpers.IAnomalyCalculation) string {
	distances := make([]float64, len(interfaces))
	nextBestHopOrbit, nextBestHopId := helpers.GetOrbitAndSatelliteId(nextBestHop)

	for i := range interfaces {
		if interfaces[i] == "" {
			distances[i] = math.Inf(1)
		} else {
			interfaceOrbit, interfaceId := helpers.GetOrbitAndSatelliteId(interfaces[i])
			distances[i] = anomalyCalculation.CalculateDistanceBySatelliteId(nextBestHopId, nextBestHopOrbit, interfaceId, interfaceOrbit, 0.001*timeStamp)
		}
	}

	minDistance := slices.Min(distances)

	if minDistance == math.Inf(1) {
		return ""
	}

	// Find index of minimum distance
	for i := range distances {
		if distances[i] == minDistance {
			return interfaces[i]
		}
	}

	return ""
}
