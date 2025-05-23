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
			gridPlus[4*(orbit*numberOfSatellitesPerOrbit+satellite)] = Pair{
				Id:              0,
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbit, nextIdInOrbit),
			}
			gridPlus[4*(orbit*numberOfSatellitesPerOrbit+satellite)+1] = Pair{
				Id:              1,
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbit, previousIdInOrbit),
			}
			gridPlus[4*(orbit*numberOfSatellitesPerOrbit+satellite)+2] = Pair{
				Id:              2,
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbitOnLeft, satellite),
			}
			gridPlus[4*(orbit*numberOfSatellitesPerOrbit+satellite)+3] = Pair{
				Id:              3,
				FirstSatellite:  fmt.Sprintf("%s-%d-%d", consellationName, orbit, satellite),
				SecondSatellite: fmt.Sprintf("%s-%d-%d", consellationName, orbitOnRight, satellite),
			}
		}
	}

	return gridPlus
}
