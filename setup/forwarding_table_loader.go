package setup

import (
	"encoding/csv"
	"os"
	"strconv"

	"Starfield/actors"
)

func LoadForwardingTableInMemory(forwardingFileName string) map[int]actors.ForwardingEntry {
	forwardingTable := make(map[int]actors.ForwardingEntry)

	file, err := os.Open(forwardingFileName)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)

	// ignore the header
	_, _ = reader.Read()
	// read the data
	records, _ := reader.ReadAll()

	for _, record := range records {
		timeStamp, _ := strconv.Atoi(record[0])
		if forwardingTable[timeStamp] == nil {
			forwardingTable[timeStamp] = make(actors.ForwardingEntry)
		}
		forwardingTable[timeStamp][record[1]] = record[2]
	}

	return forwardingTable
}
