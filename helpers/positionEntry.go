package helpers

import (
	"fmt"
	"math"
)

type PositionEntryList []IPositionEntry

type PositionEntry struct {
	TimeStamp int
	Id        string
	Latitude  float64
	Longitude float64
	Radius    float64
}

type IPositionEntry interface {
	getHeaders() []string
	toSlice() []string
	GetTimeStamp() int
}

func (entry *PositionEntry) GetTimeStamp() int {
	return entry.TimeStamp
}

func (entry *PositionEntry) getHeaders() []string {
	return []string{"TimeStamp(ms)", "Id", "Latitude", "Longitude", "Radius(m)"}
}

func (entry *PositionEntry) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", entry.TimeStamp),
		entry.Id,
		fmt.Sprintf("%f", entry.Latitude*180.0/math.Pi),
		fmt.Sprintf("%f", entry.Longitude*180.0/math.Pi),
		fmt.Sprintf("%f", entry.Radius),
	}
}

func GetRowsFromPositionEntries(entries *PositionEntryList) [][]string {
	var rows [][]string
	for i, entry := range *entries {
		if i == 0 {
			rows = append(rows, entry.getHeaders())
		}
		rows = append(rows, entry.toSlice())
	}
	return rows
}
