package routing

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/shayunak/SatSimGo/helpers"
)

func getOrbitAndSatelliteId(satelliteName string) (int, int) {
	splitted := strings.Split(satelliteName, "-")
	orbit, _ := strconv.Atoi(splitted[1])
	id, _ := strconv.Atoi(splitted[2])

	return orbit, id
}

func DijkstraModifiedOnGridPlus(nextBestHop string, timeStamp int, interfaces []string, anomalyCalculation helpers.IAnomalyCalculation) int {
	distances := make([]float64, len(interfaces))
	nextBestHopOrbit, nextBestHopId := getOrbitAndSatelliteId(nextBestHop)

	for i := 0; i < len(interfaces); i++ {
		if interfaces[i] == "" {
			distances[i] = math.Inf(1)
		} else {
			interfaceOrbit, interfaceId := getOrbitAndSatelliteId(interfaces[i])
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
