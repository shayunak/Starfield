package setup

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/shayunak/SatSimGo/actors"
)

func openTrafficGeneratorFile(fileName string) (*os.File, *csv.Reader) {
	trafficGeneratorFilePath := "./input/" + fileName
	file, err := os.Open(trafficGeneratorFilePath)
	if err != nil {
		panic(err)
	}

	csvReader := csv.NewReader(file)

	_, err = csvReader.Read()
	if err != nil {
		panic(err)
	}

	return file, csvReader
}

// Generator file format: Timestamp, Source, Destination, Length
func readTrafficGeneratorFile(generatorFile string) map[string][]actors.TrafficEntry {
	trafficMatrix := make(map[string][]actors.TrafficEntry)

	file, reader := openTrafficGeneratorFile(generatorFile)

	defer file.Close()

	// read the data
	records, _ := reader.ReadAll()

	for _, record := range records {
		timeStamp, _ := strconv.Atoi(record[0])
		source := record[1]
		destination := record[2]
		length, _ := strconv.ParseFloat(record[3], 64)
		trafficEntry := actors.TrafficEntry{
			Destination: destination,
			TimeStamp:   timeStamp,
			Length:      length,
		}
		trafficMatrix[source] = append(trafficMatrix[source], trafficEntry)
	}

	return trafficMatrix
}

func loadTrafficOnNodes(generatorFile string, groundStations *GroundStationList, maxPacketSize float64) int {
	trafficMatrix := readTrafficGeneratorFile(generatorFile)
	totalNumberOfPackets := 0
	packetId := 0

	for _, gs := range *groundStations {
		sourceEntry, isPresent := trafficMatrix[gs.GetName()]
		if isPresent {
			numberOfPackets, newId := gs.GenerateTraffic(packetId, sourceEntry, maxPacketSize)
			packetId = newId
			totalNumberOfPackets += numberOfPackets
		}
	}

	return totalNumberOfPackets
}
