package helpers

import (
	"fmt"
)

type DistanceEntryList []IDistanceEntry

type DistanceEntry struct {
	TimeStamp  int
	FromDevice string
	ToDevice   string
	Distance   int
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
	return []string{"TimeStamp(ms)", "FirstDeviceId", "SecondDeviceId", "Distance(m)"}
}

func (entry *DistanceEntry) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", entry.TimeStamp),
		entry.FromDevice,
		entry.ToDevice,
		fmt.Sprintf("%d", entry.Distance),
	}
}

func GetRowsFromDistanceEntries(entries *DistanceEntryList) [][]string {
	var rows [][]string
	for i, entry := range *entries {
		if i == 0 {
			rows = append(rows, entry.getHeaders())
		}
		rows = append(rows, entry.toSlice())
	}
	return rows
}
