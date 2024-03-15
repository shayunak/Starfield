package actors

import (
	"github.com/umpc/go-sortedmap"
)

type UpdatePoisitionMessage struct {
	SatelliteId string
	Position    CartesianCoordinates
	TimeStamp   int
}

type SpaceSatelliteChannel chan UpdatePoisitionMessage

type SpaceSatelliteChannels []*SpaceSatelliteChannel

type EventMapRecord sortedmap.Record

type EventMap *sortedmap.SortedMap

type ISpace interface {
	Run()
	SetSatelliteChannels(channels SpaceSatelliteChannels)
}

type Event struct {
	TimeStamp int
	Id        string
	X         float64
	Y         float64
	Z         float64
}

type Space struct {
	TimeStamp              int
	TotalSimulationTime    int
	SpaceSatelliteChannels SpaceSatelliteChannels
	Events                 EventMap
}

func (space Space) Run() {

}

func (space Space) SetSatelliteChannels(channels SpaceSatelliteChannels) {
	space.SpaceSatelliteChannels = channels
}
