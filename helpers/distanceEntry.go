package helpers

import (
	"fmt"
)

type DistanceEntryList []IDistanceEntry

type DistanceEntry struct {
	TimeStamp         int
	FirstSatelliteId  string
	SecondSatelliteId string
	Distance          int
}

type IDistanceEntry interface {
	getHeaders() []string
	toSlice() []string
	GetTimeStamp() int
}

func (entry *DistanceEntry) GetTimeStamp() int {
	return entry.TimeStamp
}

func (entry *DistanceEntry) getHeaders() []string {
	return []string{"TimeStamp", "FirstSatelliteId", "SecondSatelliteId", "Distance"}
}

func (entry *DistanceEntry) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", entry.TimeStamp),
		entry.FirstSatelliteId,
		entry.SecondSatelliteId,
		fmt.Sprintf("%d", entry.Distance),
	}
}

func GetRowsFromEvents(entries *DistanceEntryList) [][]string {
	var rows [][]string
	for i, event := range *entries {
		if i == 0 {
			rows = append(rows, event.getHeaders())
		}
		rows = append(rows, event.toSlice())
	}
	return rows
}
