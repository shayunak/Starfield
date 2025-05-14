package setup

import (
	"encoding/csv"
	"os"

	"github.com/shayunak/SatSimGo/connections"
)

func openISLTopologyFile(fileName string) (*os.File, *csv.Reader) {
	ISLTopologyFilePath := "./input/" + fileName
	file, err := os.Open(ISLTopologyFilePath)
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

// ISL file format: SatelliteOne, SatelliteTwo
// Important: The file should have both (S1, S2) and (S2, S1) pairs symmetrically, like a matrix
func readISLTopologyFile(ISLTopologyFileName string) [][]string {
	file, reader := openISLTopologyFile(ISLTopologyFileName)

	defer file.Close()

	// read the data
	records, _ := reader.ReadAll()

	return records
}

func GenerateISLTopology(ISLTopologyFileName string) []connections.Pair {
	satelliteIds := make(map[string]int)
	ISLRecords := readISLTopologyFile(ISLTopologyFileName)
	topology := make([]connections.Pair, len(ISLRecords))

	for indx, record := range ISLRecords {
		firstSatellite := record[0]
		secondSatellite := record[1]
		pairId, ok := satelliteIds[firstSatellite]
		if !ok {
			pairId = 0
			satelliteIds[firstSatellite] = 1
		} else {
			satelliteIds[firstSatellite] = pairId + 1
		}
		topology[indx] = connections.Pair{
			Id:              pairId,
			FirstSatellite:  firstSatellite,
			SecondSatellite: secondSatellite,
		}
	}
	return topology
}
