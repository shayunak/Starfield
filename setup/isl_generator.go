package setup

import (
	"encoding/csv"
	"os"
	"strconv"

	"Starfield/connections"
)

func openISLTopologyFile(fileName string) (*os.File, *csv.Reader, bool) {
	ISLTopologyFilePath := "./input/" + fileName
	file, err := os.Open(ISLTopologyFilePath)
	if err != nil {
		panic(err)
	}

	csvReader := csv.NewReader(file)

	header, err := csvReader.Read()
	if err != nil {
		panic(err)
	}

	if header[0] == "TimeStamp(ms)" {
		return file, csvReader, true
	}
	return file, csvReader, false
}

// ISL file format: (TimeStamp(ms)), FirstSatellite, SecondSatellite
// Important: The file should have both (S1, S2) and (S2, S1) pairs symmetrically, like a matrix
func readISLTopologyFile(ISLTopologyFileName string) [][]string {
	var records [][]string
	file, reader, isDynamic := openISLTopologyFile(ISLTopologyFileName)

	defer file.Close()

	if isDynamic {
		for {
			record, err := reader.Read()
			if err != nil {
				break
			}
			timeStamp, _ := strconv.Atoi(record[0])
			if timeStamp > 0 {
				break
			}
			records = append(records, []string{record[1], record[2]})
		}
	} else {
		records, _ = reader.ReadAll()
	}

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
