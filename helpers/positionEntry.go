package helpers

import (
	"fmt"
	"math"
)

type PositionEntryList []IPositionEntry

type SphericalPositionEntry struct {
	TimeStamp int
	Id        string
	Latitude  float64
	Longitude float64
	Radius    float64
}

type CartesianPositionEntry struct {
	TimeStamp int
	Id        string
	X         float64
	Y         float64
	Z         float64
}

type IPositionEntry interface {
	getHeaders() []string
	toSlice() []string
	GetTimeStamp() int
}

func (entry *SphericalPositionEntry) GetTimeStamp() int {
	return entry.TimeStamp
}

func (entry *SphericalPositionEntry) getHeaders() []string {
	return []string{"TimeStamp(ms)", "Id", "Latitude(deg)", "Longitude(deg)", "Radius(m)"}
}

func (entry *SphericalPositionEntry) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", entry.TimeStamp),
		entry.Id,
		fmt.Sprintf("%f", entry.Latitude*180.0/math.Pi),
		fmt.Sprintf("%f", entry.Longitude*180.0/math.Pi),
		fmt.Sprintf("%f", entry.Radius),
	}
}

func (entry *CartesianPositionEntry) GetTimeStamp() int {
	return entry.TimeStamp
}

func (entry *CartesianPositionEntry) getHeaders() []string {
	return []string{"TimeStamp(ms)", "Id", "X(m)", "Y(m)", "Z(m)"}
}

func (entry *CartesianPositionEntry) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", entry.TimeStamp),
		entry.Id,
		fmt.Sprintf("%f", entry.X),
		fmt.Sprintf("%f", entry.Y),
		fmt.Sprintf("%f", entry.Z),
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
