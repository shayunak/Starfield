package setup

import (
	"math"
	"sync"

	"github.com/shayunak/SatSimGo/actors"
	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
)

func initCalculators(config Config) (helpers.IAnomalyCalculation, helpers.IGroundStationCalculation) {
	inclinationRadians := config.OrbitConfig.Inclination * math.Pi / 180.0
	minAscensionAngle := config.OrbitConfig.MinAscensionAngle
	maxAscensionAngle := config.OrbitConfig.MaxAscensionAngle
	numberOfOrbits := config.OrbitConfig.NumberOfOrbits
	numberOfSatellitesPerOrbit := config.OrbitConfig.NumberOfSatellitesPerOrbit
	orbitRadius := config.OrbitConfig.EarthRadius + config.OrbitConfig.Altitude
	anomalyStep := 360.0 / float64(numberOfSatellitesPerOrbit)
	weatherRadius := config.OrbitConfig.EarthRadius + config.OrbitConfig.MinAltitudeISL
	maxIslLenght := 2 * math.Sqrt(math.Pow(orbitRadius, 2)-math.Pow(weatherRadius, 2))
	meanMotionRadiansPerSecond := config.SatelliteConfig.MeanMotionRevPerDay * ((2.0 * math.Pi) / (24.0 * 60.0 * 60.0))
	earthMotionRadiansPerSecond := config.OrbitConfig.EarthRotationPeriod * ((2.0 * math.Pi) / (24.0 * 60.0 * 60.0))
	ascensionStep := 0.0
	if numberOfOrbits > 1 {
		ascensionStep = (maxAscensionAngle - minAscensionAngle) / float64(numberOfOrbits-1)
	}

	orbitalCalc := &helpers.OrbitalCalculations{
		InclinationSinus:   math.Sin(inclinationRadians),
		InclinationCosinus: math.Cos(inclinationRadians),
		AscensionStep:      ascensionStep * (math.Pi / 180.0),
		NumberOfOrbits:     numberOfOrbits,
		MinAscensionAngle:  minAscensionAngle * math.Pi / 180.0,
		MaxAscensionAngle:  maxAscensionAngle * math.Pi / 180.0,
	}

	anomalyCalc := &helpers.AnomalyCalculations{
		ConsellationName:           config.ConsellationName,
		LengthLimitRatio:           1.0 - math.Pow(maxIslLenght/orbitRadius, 2)/2,
		MaxDistance:                maxIslLenght,
		NumberOfSatellitesPerOrbit: numberOfSatellitesPerOrbit,
		AnomalyStep:                anomalyStep * (math.Pi / 180.0),
		MeanMotion:                 meanMotionRadiansPerSecond,
		Radius:                     orbitRadius,
		OrbitalCalculations:        orbitalCalc,
		PhaseDiffEnabled:           config.OrbitConfig.PhaseDiffEnabled,
	}

	groundCalc := &helpers.GroundStationCalculation{
		AnomalyCalculations: anomalyCalc,
		ElevationLimitRatio: calculateElevationLimitRatio(config.OrbitConfig.EarthRadius, orbitRadius,
			config.SatelliteConfig.MinElevationAngle, config.OrbitConfig.Altitude),
		Altitude:           config.OrbitConfig.Altitude,
		EarthOrbitRatio:    config.OrbitConfig.EarthRadius / orbitRadius,
		EarthRotaionMotion: earthMotionRadiansPerSecond,
	}

	return anomalyCalc, groundCalc
}

func initSpace(space *actors.ISpace, config Config, timeStep int, totalSimulationTime int) {
	*space = &actors.Space{
		TotalSimulationTime:    totalSimulationTime,
		SpaceSatelliteChannels: nil,
		DistancesSpaceChannels: nil,
		SatelliteNames:         nil,
		DistanceEntries:        make(helpers.DistanceEntryList, 0),
		ConsellationName:       config.ConsellationName,
		TimeStep:               timeStep,
		TimeStamp:              0,
	}
}

func initTopology(satellites SatelliteList, entries map[string]map[string]connections.InterfaceEntry) {
	for _, satellite := range satellites {
		for _, entry := range entries[satellite.GetName()] {
			satellite.AddISLConnectionOnId(entry.InterfaceId, entry.ConnectedDevice, entry.ReceiveChannel, entry.SendChannel)
		}
	}
}

func SetupSimulatorDistances(configFileName string, groundStationFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var space actors.ISpace

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initSpace(&space, config, timeStep, totalSimulationTime)
	groundStationSpecs := initGroundStations(&groundStations, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundStationSpecs)

	// starting the actors
	channels := startDistancesSatellites(satellites)
	channels = append(channels, startDistancesGroundStations(groundStations)...)

	space.SetDistancesDeviceChannels(&channels)
	space.RunDistances(simulationDone)
}

func SetupForwardingSimulation(configFileName string, groundStationFileName string, trafficFile string, forwardingFolder string, timeStep int,
	totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var space actors.ISpace

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initSpace(&space, config, timeStep, totalSimulationTime)
	groundStationSpecs := initGroundStations(&groundStations, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundStationSpecs)

	// reading the traffic file
	loadTrafficOnNodes(trafficFile, &satellites, config.SatelliteConfig.MaxPacketSize)

	// adding forwarding file data to satellites
	for _, satellite := range satellites {
		satellite.SetForwardingFile(forwardingFolder)
	}

	// bringing up the ISL topology
	topologyPairs := connections.GenerateGridPlus(config.OrbitConfig.NumberOfOrbits, config.OrbitConfig.NumberOfSatellitesPerOrbit, config.ConsellationName)
	topologyList := connections.GetTopologyList(topologyPairs, config.SatelliteConfig.InterfaceBufferSize)

	// adding topology data to satellites
	initTopology(satellites, topologyList)

	// starting the actors
	channels, satelliteNames := startSatellites(satellites)
	space.SetSatelliteChannels(&channels, satelliteNames)
	space.Run(simulationDone)
}
