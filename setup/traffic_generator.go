package setup

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/shayunak/SatSimGo/actors"
)

func readTrafficGeneratorFile(generatorFile string) map[string][]actors.TrafficEntry {
	trafficMatrix := make(map[string][]actors.TrafficEntry)

	file, err := os.Open(generatorFile)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)

	// read the data
	records, _ := reader.ReadAll()

	for _, record := range records {
		source := record[0]
		destination := record[1]
		timeStamp, _ := strconv.Atoi(record[2])
		length, _ := strconv.Atoi(record[3])
		trafficEntry := actors.TrafficEntry{
			Destination: destination,
			TimeStamp:   timeStamp,
			Length:      length,
		}
		trafficMatrix[source] = append(trafficMatrix[source], trafficEntry)
	}

	return trafficMatrix
}

func loadTrafficOnNodes(generatorFile string, satellites *SatelliteList) {
	trafficMatrix := readTrafficGeneratorFile(generatorFile)

	for _, satellite := range *satellites {
		satellite.GenerateTraffic(trafficMatrix[satellite.GetName()])
	}
}
