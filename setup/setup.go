package setup

import (
	"math"
	"sync"

	"github.com/shayunak/SatSimGo/actors"
	"github.com/shayunak/SatSimGo/connections"
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

			*satellites = append(*satellites, actors.NewSatellite(satellite, anomaly, timeStep, totalSimulationTimeMilliseconds,
				orbit, anomalyCalc, config.SatelliteConfig.NumberOfISLs, config.SatelliteConfig.NumberOfGSLs, config.SatelliteConfig.SpeedOfLightVac,
				config.SatelliteConfig.ISLBandwidth, config.SatelliteConfig.ISLLinkNoiseCoef, config.SatelliteConfig.ISLAcquisitionTime))

		}
	}
}

func initSpace(space *actors.ISpace, config Config, timeStep int, totalSimulationTime int) {
	*space = &actors.Space{
		TotalSimulationTime:             totalSimulationTime,
		SpaceSatelliteChannels:          nil,
		DistancesSpaceSatelliteChannels: nil,
		SatelliteNames:                  nil,
		DistanceEntries:                 make(helpers.DistanceEntryList, 0),
		ConsellationName:                config.ConsellationName,
		TimeStep:                        timeStep,
		TimeStamp:                       0,
	}
}

func initTopology(satellites SatelliteList, entries map[string]map[string]connections.InterfaceEntry) {
	for _, satellite := range satellites {
		for _, entry := range entries[satellite.GetName()] {
			satellite.AddISLConnection(entry.ConnectedDevice, entry.ReceiveChannel, entry.SendChannel)
		}
	}
}

func startDistancesSatellites(satellites SatelliteList) *actors.DistanceSpaceSatelliteChannels {
	channels := make(actors.DistanceSpaceSatelliteChannels, 0)
	for _, satellite := range satellites {
		channel := make(actors.DistanceSpaceSatelliteChannel)
		channels = append(channels, &channel)
		satellite.SetDistanceSpaceChannel(&channel)
		satellite.RunDistances()
	}
	return &channels
}

func startSatellites(satellites SatelliteList) (*actors.SpaceSatelliteChannels, []string) {
	channels := make(actors.SpaceSatelliteChannels, 0)
	satelliteNames := make([]string, 0)
	for _, satellite := range satellites {
		channel := make(actors.SpaceSatelliteChannel)
		channels = append(channels, &channel)
		satelliteNames = append(satelliteNames, satellite.GetName())
		satellite.SetSpaceChannel(&channel)
		satellite.Run()
	}
	return &channels, satelliteNames
}

func SetupSimulatorDistances(configFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var space actors.ISpace

	// reading the config file
	config := getConfig(configFileName)

	// initializing the actors
	initSpace(&space, config, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, timeStep, totalSimulationTime)

	// starting the actors
	space.SetDistancesSatelliteChannels(startDistancesSatellites(satellites))
	space.RunDistances(simulationDone)
}

func SetupForwardingSimulation(configFileName string, trafficFile string, forwardingFolder string, timeStep int,
	totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var space actors.ISpace

	// reading the config file
	config := getConfig(configFileName)

	// initializing the actors
	initSpace(&space, config, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, timeStep, totalSimulationTime)

	// reading the traffic file
	loadTrafficOnNodes(trafficFile, &satellites, config.SatelliteConfig.MaxPacketSize)

	// adding forwarding file data to satellites
	for _, satellite := range satellites {
		satellite.SetForwardingFile(forwardingFolder)
	}

	// bringing up the ISL topology
	topologyPairs := connections.GenerateGridPlus(config.OrbitConfig.NumberOfOrbits, config.OrbitConfig.NumberOfSatellitesPerOrbit, config.ConsellationName)
	topologyList := connections.GetTopologyList(topologyPairs)

	// adding topology data to satellites
	initTopology(satellites, topologyList)

	// starting the actors
	space.SetSatelliteChannels(startSatellites(satellites))
	space.Run(simulationDone)
}
