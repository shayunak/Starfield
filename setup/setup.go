package setup

import (
	"math"
	"sync"

	"github.com/shayunak/SatSimGo/actors"
	"github.com/shayunak/SatSimGo/helpers"
)

type SatelliteList []actors.ISatellite

func initSatellites(satellites *SatelliteList, config Config, timeStep int, totalSimulationTime int) {
	minAscensionAngle := config.OrbitConfig.MinAscensionAngle
	maxAscensionAngle := config.OrbitConfig.MaxAscensionAngle
	numberOfOrbits := config.OrbitConfig.NumberOfOrbits
	numberOfSatellitesPerOrbit := config.OrbitConfig.NumberOfSatellitesPerOrbit
	inclinationRadians := config.OrbitConfig.Inclination * math.Pi / 180.0
	orbit_radius := config.OrbitConfig.EarthRadius + config.OrbitConfig.Altitude
	weather_radius := config.OrbitConfig.EarthRadius + config.OrbitConfig.MinAltitudeISL
	maxIslLenght := 2 * math.Sqrt(math.Pow(orbit_radius, 2)-math.Pow(weather_radius, 2))
	ascensionStep := (maxAscensionAngle - minAscensionAngle) / float64(numberOfOrbits)
	anomalyStep := 360.0 / float64(numberOfSatellitesPerOrbit)
	meanMotionRadiansPerSecond := config.SatelliteConfig.MeanMotionRevPerDay * ((2.0 * math.Pi) / (24.0 * 60.0 * 60.0))
	totalSimulationTimeMilliseconds := totalSimulationTime * 1000 // in milliseconds

	orbitalCalc := &helpers.OrbitalCalculations{
		InclinationSinus:   math.Sin(inclinationRadians),
		InclinationCosinus: math.Cos(inclinationRadians),
		LengthLimitRatio:   1.0 - math.Pow(maxIslLenght/orbit_radius, 2)/2,
		AscensionStep:      ascensionStep * (math.Pi / 180.0),
		NumberOfOrbits:     numberOfOrbits,
		MinAscensionAngle:  minAscensionAngle * math.Pi / 180.0,
		MaxAscensionAngle:  maxAscensionAngle * math.Pi / 180.0,
	}

	anomalyCalc := &helpers.AnomalyCalculations{
		ConsellationName:           config.ConsellationName,
		LengthLimitRatio:           1.0 - math.Pow(maxIslLenght/orbit_radius, 2)/2,
		NumberOfSatellitesPerOrbit: numberOfSatellitesPerOrbit,
		AnomalyStep:                anomalyStep * (math.Pi / 180.0),
		MeanMotion:                 meanMotionRadiansPerSecond,
		Radius:                     orbit_radius,
		OrbitalCalculations:        orbitalCalc,
		PhaseDiffEnabled:           config.OrbitConfig.PhaseDiffEnabled,
	}

	for orbit := 0; orbit < numberOfOrbits; orbit++ {
		ascensionNodeDegree := minAscensionAngle + float64(orbit)*ascensionStep
		phase_shift := 0.0

		if config.OrbitConfig.PhaseDiffEnabled && orbit%2 == 1 {
			phase_shift = anomalyStep / 2.0
		}

		orbit := helpers.NewOrbit(orbit_radius, config.OrbitConfig.Altitude, ascensionNodeDegree,
			inclinationRadians, orbit, config.ConsellationName, phase_shift)

		for satellite := 0; satellite < numberOfSatellitesPerOrbit; satellite++ {
			anomaly := phase_shift + float64(satellite)*anomalyStep

			*satellites = append(*satellites, actors.NewSatellite(satellite, anomaly, timeStep, totalSimulationTimeMilliseconds, orbit, anomalyCalc))

		}
	}
}

func initSpace(space *actors.ISpace, config Config, timeStep int, totalSimulationTime int) {
	*space = &actors.Space{
		TotalSimulationTime:    totalSimulationTime,
		SpaceSatelliteChannels: nil,
		Events:                 make(actors.EventList, 0),
		ConsellationName:       config.ConsellationName,
		TimeStep:               timeStep,
		TimeStamp:              0,
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
	initSatellites(&satellites, config, timeStep, totalSimulationTime)
	initSpace(&space, config, timeStep, totalSimulationTime)

	// starting the actors
	space.SetSatelliteChannels(startSatellites(satellites))
	space.Run(simulationDone)
}
