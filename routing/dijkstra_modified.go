package routing

import (
	"math"
	"slices"

	"github.com/shayunak/SatSimGo/helpers"
)

func DijkstraModifiedOnGridPlus(nextBestHop string, timeStamp int, interfaces []string, anomalyCalculation helpers.IAnomalyCalculation) int {
	distances := make([]float64, len(interfaces))
	nextBestHopOrbit, nextBestHopId := helpers.GetOrbitAndSatelliteId(nextBestHop)

	for i := 0; i < len(interfaces); i++ {
		if interfaces[i] == "" {
			distances[i] = math.Inf(1)
		} else {
			interfaceOrbit, interfaceId := helpers.GetOrbitAndSatelliteId(interfaces[i])
			distances[i] = anomalyCalculation.CalculateDistanceBySatelliteId(nextBestHopId, nextBestHopOrbit, interfaceId, interfaceOrbit, 0.001*float64(timeStamp))
		}
	}

	minDistance := slices.Min(distances)

	if minDistance == math.Inf(1) {
		return -1
	}

	// Find index of minimum distance
	for i := 0; i < len(distances); i++ {
		if distances[i] == minDistance {
			return i
		}
	}

	return -1
}
