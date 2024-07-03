package connections

import (
	"fmt"
)

// default initial ISL topology of the satellites

type Pair struct {
	FirstSatellite  string
	SecondSatellite string
	Channel         chan Packet
}

type InterfaceEntry struct {
	ConnectedDevice string
	SendChannel     *chan Packet
	ReceiveChannel  *chan Packet
}

func GenerateGridPlus(numberOfOrbits int, numberOfSatellitesPerOrbit int, consellationName string) []Pair {
	gridPlus := make([]Pair, 4*numberOfOrbits*numberOfSatellitesPerOrbit)

	for orbit := 0; orbit < numberOfOrbits; orbit++ {
		orbitOnLeft := (orbit + numberOfOrbits - 1) % numberOfOrbits
		orbitOnRight := (orbit + 1) % numberOfOrbits
		for satellite := 0; satellite < numberOfSatellitesPerOrbit; satellite++ {
			nextIdInOrbit := (satellite + 1) % numberOfSatellitesPerOrbit
			previousIdInOrbit := (satellite + numberOfSatellitesPerOrbit - 1) % numberOfSatellitesPerOrbit
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbit, nextIdInOrbit),
				Channel:         make(chan Packet),
			}
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite+1] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbit, previousIdInOrbit),
				Channel:         make(chan Packet),
			}
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite+2] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbitOnLeft, satellite),
				Channel:         make(chan Packet),
			}
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite+3] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbitOnRight, satellite),
				Channel:         make(chan Packet),
			}
		}
	}

	return gridPlus
}

func GetTopologyList(pairs []Pair) map[string]map[string]InterfaceEntry {
	topologyList := make(map[string]map[string]InterfaceEntry)

	// First pass initializing the matrix
	for _, pair := range pairs {
		if topologyList[pair.FirstSatellite] == nil {
			topologyList[pair.FirstSatellite] = make(map[string]InterfaceEntry)
			if topologyList[pair.SecondSatellite] == nil {
				topologyList[pair.FirstSatellite][pair.SecondSatellite] = InterfaceEntry{
					ConnectedDevice: pair.SecondSatellite,
					SendChannel:     &pair.Channel,
					ReceiveChannel:  nil,
				}
			}
		}
	}

	// Second pass assigning recieve channels
	for _, pair := range pairs {
		entry := topologyList[pair.SecondSatellite][pair.FirstSatellite]
		topologyList[pair.SecondSatellite][pair.FirstSatellite] = InterfaceEntry{
			ConnectedDevice: entry.ConnectedDevice,
			SendChannel:     entry.SendChannel,
			ReceiveChannel:  &pair.Channel,
		}
	}

	return topologyList
}
