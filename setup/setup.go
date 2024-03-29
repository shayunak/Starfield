package setup

import (
	"fmt"
	"sync"

	"github.com/shayunak/SatSimGo/actors"
)

type SatelliteList []actors.ISatellite

func initSatellites(satellites *SatelliteList, config Config, timeStep int) {
	minAscensionAngle := config.OrbitConfig.MinAscensionAngle
	maxAscensionAngle := config.OrbitConfig.MaxAscensionAngle
	numberOfOrbits := config.OrbitConfig.NumberOfOrbits
	numberOfSatellitesPerOrbit := config.OrbitConfig.NumberOfSatellitesPerOrbit

	for orbit := 0; orbit < numberOfOrbits; orbit++ {
		ascensionNodeDegree := minAscensionAngle + float64(orbit)*(maxAscensionAngle-minAscensionAngle)/float64(numberOfOrbits)

		phase_shift := 0.0
		if config.OrbitConfig.PhaseDiffEnabled && orbit%2 == 1 {
			phase_shift = 360.0 / (2.0 * float64(numberOfSatellitesPerOrbit))
		}

		for satellite := 0; satellite < numberOfSatellitesPerOrbit; satellite++ {
			anomaly := phase_shift + float64(satellite)*360.0/float64(numberOfSatellitesPerOrbit)
			id := fmt.Sprintf("%s-%d-%d", config.ConsellationName, orbit, satellite)

			*satellites = append(*satellites, actors.NewSatellite(id, config.OrbitConfig.Altitude,
				config.OrbitConfig.EarthRadius, config.SatelliteConfig.MeanMotionRevPerDay,
				anomaly, config.OrbitConfig.Inclination, ascensionNodeDegree, timeStep))

		}
	}
}

func initSpace(space *actors.ISpace, totalSimulationTime int, config Config, timeStep int) {
	*space = &actors.Space{
		TotalSimulationTime:    totalSimulationTime * 1000, // to milliseconds
		SpaceSatelliteChannels: nil,
		Events:                 make(actors.EventList, 0),
		ConsellationName:       config.ConsellationName,
		TimeStep:               timeStep,
	}
}

func startSatellites(satellites SatelliteList) *actors.SpaceSatelliteChannels {
	channels := make(actors.SpaceSatelliteChannels, 0)
	for _, satellite := range satellites {
		channels = append(channels, satellite.GetSpaceChannel())
		satellite.Run()
	}
	return &channels
}

func SetupSimulator(configFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var space actors.ISpace

	// reading the config file
	config := getConfig(configFileName)

	// initializing the actors
	initSatellites(&satellites, config, timeStep)
	initSpace(&space, totalSimulationTime, config, timeStep)

	// starting the actors
	space.SetSatelliteChannels(startSatellites(satellites))
	space.Run(simulationDone)
}
