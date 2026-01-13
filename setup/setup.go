package setup

import (
	"fmt"
	"math"
	"os"
	"sync"

	"SatSimGo/actors"

	"SatSimGo/connections"

	"SatSimGo/helpers"
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
		ascensionStep = (maxAscensionAngle - minAscensionAngle) / float64(numberOfOrbits)
	}

	orbitalCalc := &helpers.OrbitalCalculations{
		InclinationSinus:   math.Sin(inclinationRadians),
		InclinationCosinus: math.Cos(inclinationRadians),
		AscensionStep:      ascensionStep * (math.Pi / 180.0),
		NumberOfOrbits:     numberOfOrbits,
		MinAscensionAngle:  minAscensionAngle * math.Pi / 180.0,
		MaxAscensionAngle:  maxAscensionAngle * math.Pi / 180.0,
		UseGPU:             config.UseGPU,
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
		UseGPU:                     config.UseGPU,
	}

	groundCalc := &helpers.GroundStationCalculation{
		AnomalyCalculations: anomalyCalc,
		ElevationLimitRatio: calculateElevationLimitRatio(config.OrbitConfig.EarthRadius, orbitRadius,
			config.SatelliteConfig.MinElevationAngle, config.OrbitConfig.Altitude),
		Altitude:                    config.OrbitConfig.Altitude,
		EarthRadius:                 config.OrbitConfig.EarthRadius,
		EarthOrbitRatio:             config.OrbitConfig.EarthRadius / orbitRadius,
		EarthRotaionMotion:          earthMotionRadiansPerSecond,
		GroundStationsDistanceLimit: calculateGroundStationDistancLimit(orbitRadius, config.SatelliteConfig.MinElevationAngle, config.OrbitConfig.Altitude),
		GroundStations:              nil,
		GroundStationCalculationC:   nil,
		UseGPU:                      config.UseGPU,
	}

	return anomalyCalc, groundCalc
}

func initLogger(logger *actors.ILogger, config Config, timeStep int, totalSimulationTime int, totalNumberOfPackets int) *chan float64 {
	coordinatorChannel := make(chan float64, 1)
	*logger = &actors.Logger{
		TotalSimulationTime:         float64(totalSimulationTime) * 1000, // in milliseconds
		LoggerDeviceChannels:        nil,
		CoordinatorChannel:          &coordinatorChannel,
		DistancesLoggerChannels:     nil,
		DeviceNames:                 nil,
		DistanceEntries:             make(helpers.DistanceEntryList, 0),
		ConsellationName:            config.ConsellationName,
		RemainingUnprocessedPackets: totalNumberOfPackets,
		TimeStep:                    timeStep,
		TimeStamp:                   0,
		NumberOfOrbits:              config.OrbitConfig.NumberOfOrbits,
		NumberOfSatellitesPerOrbit:  config.OrbitConfig.NumberOfSatellitesPerOrbit,
	}
	return &coordinatorChannel
}

func initLinker(linker *actors.ILinker) {
	*linker = &actors.Linker{
		LinkIncomingRequestChannels: nil,
		LinkRelayRequestChannels:    nil,
		DeviceNames:                 nil,
		PendingConnections:          make([]actors.LinkRequest, 0),
	}
}

func initCoordinator(coordinator *actors.ICoordinator, loggerChannel *chan float64, coordinationInterval float64, totalSimulationTime int) {
	*coordinator = &actors.Coordinator{
		ProgressTokenChannels: nil,
		LoggerChannel:         loggerChannel,
		TimeStamp:             0,
		NextTimeStamp:         0,
		NumberOfAcksPerRound:  0,
		CoordinationInterval:  coordinationInterval,
		TotalSimulationTime:   float64(totalSimulationTime) * 1000.0, // in milliseconds
	}
}

func initTopology(satellites SatelliteList, entries map[string]map[string]connections.InterfaceEntry) {
	for _, satellite := range satellites {
		for _, entry := range entries[satellite.GetName()] {
			satellite.AddISLConnectionOnId(entry.InterfaceId, entry.ConnectedDevice, entry.ReceiveChannel, entry.SendChannel)
		}
	}
}

func SetupSimulatorCartesianPositions(configFileName string, groundStationFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var logger actors.ILogger

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initLogger(&logger, config, timeStep, totalSimulationTime, 0)
	initGroundStations(&groundStations, config, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundCalc)

	// starting the actors
	channels := startCartesianPositionsSatellites(satellites)
	channels = append(channels, startCartesianPositionsGroundStations(groundStations)...)

	logger.SetCartesianPositionsDeviceChannels(&channels)
	logger.RunCartesianPositions(simulationDone)
}

func SetupSimulatorSphericalPositions(configFileName string, groundStationFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var logger actors.ILogger

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initLogger(&logger, config, timeStep, totalSimulationTime, 0)
	initGroundStations(&groundStations, config, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundCalc)

	// starting the actors
	channels := startSphericalPositionsSatellites(satellites)
	channels = append(channels, startSphericalPositionsGroundStations(groundStations)...)

	logger.SetSphericalPositionsDeviceChannels(&channels)
	logger.RunSphericalPositions(simulationDone)
}

func SetupSimulatorDistances(configFileName string, groundStationFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var logger actors.ILogger

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initLogger(&logger, config, timeStep, totalSimulationTime, 0)
	initGroundStations(&groundStations, config, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundCalc)

	// starting the actors
	channels := startDistancesSatellites(satellites)
	channels = append(channels, startDistancesGroundStations(groundStations)...)

	logger.SetDistancesDeviceChannels(&channels)
	logger.RunDistances(simulationDone)
}

func SetupForwardingSimulationGridPlus(configFileName string, groundStationFileName string, trafficFile string, forwardingFolder string, timeStep int,
	totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var logger actors.ILogger
	var linker actors.ILinker
	var coordinator actors.ICoordinator

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initGroundStations(&groundStations, config, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundCalc)

	// reading the traffic file
	totalNumberOfPackets := loadTrafficOnNodes(trafficFile, &groundStations, config.SatelliteConfig.MaxPacketSize)

	// adding forwarding file data to satellites
	for _, satellite := range satellites {
		satelliteForwardingFileName := fmt.Sprintf("./forwarding_table/%s/%s.csv", forwardingFolder, satellite.GetName())
		satellite.SetForwardingTable(LoadForwardingTableInMemory(satelliteForwardingFileName))
	}

	// adding forwarding file data to ground stations
	for _, gs := range groundStations {
		groundStationForwardingFileName := fmt.Sprintf("./forwarding_table/%s/%s.csv", forwardingFolder, gs.GetName())
		if _, err := os.Stat(groundStationForwardingFileName); err == nil {
			gs.SetForwardingTable(LoadForwardingTableInMemory(groundStationForwardingFileName))
		}
	}

	// initializing the Logger and  the Linker
	coordinatorChannel := initLogger(&logger, config, timeStep, totalSimulationTime, totalNumberOfPackets)
	initLinker(&linker)
	initCoordinator(&coordinator, coordinatorChannel, config.CoordinationInterval, totalSimulationTime)

	// bringing up the ISL topology
	topologyPairs := connections.GenerateGridPlus(config.OrbitConfig.NumberOfOrbits, config.OrbitConfig.NumberOfSatellitesPerOrbit, config.ConsellationName)
	topologyList := connections.GetTopologyList(topologyPairs, config.SatelliteConfig.InterfaceBufferSize)

	// adding topology data to satellites
	initTopology(satellites, topologyList)

	// starting the actors
	satelliteLogChannels, satelliteIncomingLinkChannels, satelliteOutgoingLinkChannels, satelliteTokenChannels, satelliteAckChannels, satelliteNames := startSatellites(satellites)
	groundStationLogChannels, groundStationIncomingLinkChannels, groundStationOutgoingLinkChannels, groundStationTokenChannels, groundStationAckChannels, groundStationNames := startGroundStations(groundStations)
	logChannels := append(groundStationLogChannels, satelliteLogChannels...)
	tokenChannels := append(groundStationTokenChannels, satelliteTokenChannels...)
	ackChannels := append(groundStationAckChannels, satelliteAckChannels...)
	linkerIncomingChannels := append(groundStationOutgoingLinkChannels, satelliteOutgoingLinkChannels...)
	linkerOutgoingChannels := append(groundStationIncomingLinkChannels, satelliteIncomingLinkChannels...)
	names := append(groundStationNames, satelliteNames...)
	logger.SetDeviceChannels(&logChannels, names)
	coordinator.SetProgressTokenChannels(&tokenChannels)
	coordinator.SetAckTokenChannels(&ackChannels)
	linker.SetDeviceChannels(&linkerIncomingChannels, &linkerOutgoingChannels, names)
	logger.Run(simulationDone)
	linker.Run()
	coordinator.Run()
}

func SetupForwardingSimulation(configFileName string, groundStationFileName string, trafficFile string, forwardingFolder string,
	ISLTopologyFileName string, timeStep int, totalSimulationTime int, simulationDone *sync.WaitGroup) {
	var satellites SatelliteList
	var groundStations GroundStationList
	var logger actors.ILogger
	var coordinator actors.ICoordinator
	var linker actors.ILinker

	// reading the config file
	config := getConfig(configFileName)

	// initializing the calculators
	anomalyCalc, groundCalc := initCalculators(config)

	// initializing the actors
	initGroundStations(&groundStations, config, groundStationFileName, groundCalc, timeStep, totalSimulationTime)
	initSatellites(&satellites, config, anomalyCalc, timeStep, totalSimulationTime, groundCalc)

	// reading the traffic file
	totalNumberOfPackets := loadTrafficOnNodes(trafficFile, &groundStations, config.SatelliteConfig.MaxPacketSize)

	// adding forwarding file data to satellites
	for _, satellite := range satellites {
		satelliteForwardingFileName := fmt.Sprintf("./forwarding_table/%s/%s.csv", forwardingFolder, satellite.GetName())
		satellite.SetForwardingTable(LoadForwardingTableInMemory(satelliteForwardingFileName))
	}

	// adding forwarding file data to ground stations
	for _, gs := range groundStations {
		groundStationForwardingFileName := fmt.Sprintf("./forwarding_table/%s/%s.csv", forwardingFolder, gs.GetName())
		if _, err := os.Stat(groundStationForwardingFileName); err == nil {
			gs.SetForwardingTable(LoadForwardingTableInMemory(groundStationForwardingFileName))
		}
	}

	// initializing the Logger and the Linker
	coordinatorChannel := initLogger(&logger, config, timeStep, totalSimulationTime, totalNumberOfPackets)
	initLinker(&linker)
	initCoordinator(&coordinator, coordinatorChannel, config.CoordinationInterval, totalSimulationTime)

	// bringing up the ISL topology
	topologyPairs := GenerateISLTopology(ISLTopologyFileName)
	topologyList := connections.GetTopologyList(topologyPairs, config.SatelliteConfig.InterfaceBufferSize)

	// adding topology data to satellites
	initTopology(satellites, topologyList)

	// starting the actors
	satelliteLogChannels, satelliteIncomingLinkChannels, satelliteOutgoingLinkChannels, satelliteTokenChannels, satelliteAckChannels, satelliteNames := startSatellites(satellites)
	groundStationLogChannels, groundStationIncomingLinkChannels, groundStationOutgoingLinkChannels, groundStationTokenChannels, groundStationAckChannels, groundStationNames := startGroundStations(groundStations)
	logChannels := append(groundStationLogChannels, satelliteLogChannels...)
	tokenChannels := append(groundStationTokenChannels, satelliteTokenChannels...)
	ackChannels := append(groundStationAckChannels, satelliteAckChannels...)
	linkerIncomingChannels := append(groundStationOutgoingLinkChannels, satelliteOutgoingLinkChannels...)
	linkerOutgoingChannels := append(groundStationIncomingLinkChannels, satelliteIncomingLinkChannels...)
	names := append(groundStationNames, satelliteNames...)
	logger.SetDeviceChannels(&logChannels, names)
	coordinator.SetProgressTokenChannels(&tokenChannels)
	coordinator.SetAckTokenChannels(&ackChannels)
	linker.SetDeviceChannels(&linkerIncomingChannels, &linkerOutgoingChannels, names)
	logger.Run(simulationDone)
	linker.Run()
	coordinator.Run()
}
