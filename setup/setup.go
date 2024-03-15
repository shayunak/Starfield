package setup

import (
	"fmt"

	"github.com/shayunak/SatSimGo/actors"
)

type SatelliteList []actors.ISatellite

func initSatellites(satellites *SatelliteList, config Config) {
	minAscensionAngle := config.OrbitConfig.MinAscensionAngle
	maxAscensionAngle := config.OrbitConfig.MaxAscensionAngle
	numberOfOrbits := config.OrbitConfig.NumberOfOrbits
	numberOfSatellitesPerOrbit := config.OrbitConfig.NumberOfSatellitesPerOrbit

	for orbit := 0; orbit < numberOfOrbits; orbit++ {
		ascensionNodeDegree := minAscensionAngle + (maxAscensionAngle-minAscensionAngle)/float64(numberOfOrbits)

		phase_shift := 0.0
		if config.OrbitConfig.PhaseDiffEnabled && orbit%2 == 1 {
			phase_shift = 360.0 / (2.0 * float64(numberOfSatellitesPerOrbit))
		}

		for satellite := 0; satellite < numberOfSatellitesPerOrbit; satellite++ {
			anomaly := phase_shift + float64(satellite)*360.0/float64(numberOfSatellitesPerOrbit)
			id := fmt.Sprintf("%s-%d-%d", config.ConsellationName, orbit, satellite)

			*satellites = append(*satellites, actors.NewSatellite(id, config.OrbitConfig.Altitude,
				config.OrbitConfig.EarthRadius, config.SatelliteConfig.MeanMotionRevPerDay,
				anomaly, config.OrbitConfig.Inclination, ascensionNodeDegree))

		}
	}
}

func SetupSimulator(configFileName string) {
	var satellites SatelliteList
	config := getConfig(configFileName)

	initSatellites(&satellites, config)

	fmt.Println(satellites)
}
