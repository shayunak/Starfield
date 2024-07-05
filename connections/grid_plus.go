package connections

import (
	"fmt"
)

// default initial ISL topology of the satellites

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
			}
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite+1] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbit, previousIdInOrbit),
			}
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite+2] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbitOnLeft, satellite),
			}
			gridPlus[orbit*numberOfSatellitesPerOrbit+satellite+3] = Pair{
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbitOnRight, satellite),
			}
		}
	}

	return gridPlus
}
