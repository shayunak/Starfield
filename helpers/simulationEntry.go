package helpers

import (
	"fmt"
)

const EVENT_SENT string = "SEND"
const EVENT_RECEIVED string = "RECEIVE"
const EVENT_DELIVERED string = "DELIVERED"
const EVENT_DROPPED string = "DROP"
const EVENT_CONNECTION_ESTABLISHED string = "CONNECTION_ESTABLISHED"

type SimulationEntryList []ISimulationEntry

type SimulationEntry struct {
	TimeStamp  int
	EventType  string
	FromDevice string
	ToDevice   string
	PacketId   int
}

type ISimulationEntry interface {
	getHeaders() []string
	toSlice() []string
	GetTimeStamp() int
}

func (entry *SimulationEntry) GetTimeStamp() int {
	return entry.TimeStamp
}

func (entry *SimulationEntry) getHeaders() []string {
	return []string{"TimeStamp(ms)", "Event", "FromDevice", "ToDevice", "PacketId"}
}

func (entry *SimulationEntry) toSlice() []string {
	return []string{
		fmt.Sprintf("%d", entry.TimeStamp),
		entry.EventType,
		entry.FromDevice,
		entry.ToDevice,
		fmt.Sprintf("%d", entry.PacketId),
	}
}

func GetRowsFromEvents(events *SimulationEntryList) [][]string {
	var rows [][]string
	for i, event := range *events {
		if i == 0 {
			rows = append(rows, event.getHeaders())
		}
		rows = append(rows, event.toSlice())
	}
	return rows
}
