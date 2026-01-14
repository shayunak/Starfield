package setup

import (
	"math"

	"Starfield/actors"
	"Starfield/helpers"
)

type SatelliteList []actors.ISatellite

func calculateGroundStationDistancLimit(orbitRadius float64, minElevationAngle float64, altitude float64) float64 {
	minElevationAngleTangent := math.Tan(minElevationAngle * math.Pi / 180.0)
	altitudeOrbitRatio := altitude / orbitRadius

	elevationTerm := minElevationAngleTangent * math.Sqrt(math.Pow(minElevationAngleTangent, 2.0)+altitudeOrbitRatio*(2.0-altitudeOrbitRatio))
	nominator := altitudeOrbitRatio + math.Pow(minElevationAngleTangent, 2.0) - elevationTerm
	denominator := 1.0 + math.Pow(minElevationAngleTangent, 2.0)

	return orbitRadius * math.Sqrt(2.0*nominator/denominator)
}

func initSatellites(satellites *SatelliteList, config Config, anomalyCalc helpers.IAnomalyCalculation,
	timeStep int, totalSimulationTime int, groundCalc helpers.IGroundStationCalculation) {
	minAscensionAngle := config.OrbitConfig.MinAscensionAngle
	maxAscensionAngle := config.OrbitConfig.MaxAscensionAngle
	numberOfOrbits := config.OrbitConfig.NumberOfOrbits
	numberOfSatellitesPerOrbit := config.OrbitConfig.NumberOfSatellitesPerOrbit
	inclinationRadians := config.OrbitConfig.Inclination * math.Pi / 180.0
	orbitRadius := config.OrbitConfig.EarthRadius + config.OrbitConfig.Altitude
	anomalyStep := 360.0 / float64(numberOfSatellitesPerOrbit)
	totalSimulationTimeMilliseconds := float64(totalSimulationTime) * 1000 // in milliseconds
	earthMotionRadiansPerSecond := config.OrbitConfig.EarthRotationPeriod * ((2.0 * math.Pi) / (24.0 * 60.0 * 60.0))
	ascensionStep := 0.0
	if numberOfOrbits > 1 {
		ascensionStep = (maxAscensionAngle - minAscensionAngle) / float64(numberOfOrbits)
	}

	for orbit := 0; orbit < numberOfOrbits; orbit++ {
		ascensionNodeDegree := minAscensionAngle + float64(orbit)*ascensionStep
		phaseShift := 0.0

		if config.OrbitConfig.PhaseDiffEnabled && orbit%2 == 1 {
			phaseShift = anomalyStep / 2.0
		}

		orbit := helpers.NewOrbit(orbitRadius, earthMotionRadiansPerSecond, config.OrbitConfig.Altitude, ascensionNodeDegree,
			inclinationRadians, orbit, config.ConsellationName, phaseShift)

		for satellite := 0; satellite < numberOfSatellitesPerOrbit; satellite++ {
			anomaly := phaseShift + float64(satellite)*anomalyStep

			*satellites = append(*satellites, actors.NewSatellite(satellite, anomaly, float64(timeStep), totalSimulationTimeMilliseconds,
				orbit, anomalyCalc, groundCalc, config.SatelliteConfig.NumberOfISLs, config.SatelliteConfig.SpeedOfLightVac,
				config.SatelliteConfig.ISLBandwidth, config.SatelliteConfig.ISLLinkNoiseCoef, config.SatelliteConfig.GSLBandwidth,
				config.SatelliteConfig.GSLLinkNoiseCoef, config.SatelliteConfig.ISLAcquisitionTime,
				config.SatelliteConfig.MaxPacketSize, config.SatelliteConfig.InterfaceBufferSize))
		}
	}
}

func startCartesianPositionsSatellites(satellites SatelliteList) actors.CartesianPositionLoggerDeviceChannels {
	channels := make(actors.CartesianPositionLoggerDeviceChannels, 0)
	for _, satellite := range satellites {
		channel := make(actors.CartesianPositionLoggerDeviceChannel)
		channels = append(channels, &channel)
		satellite.SetCartesianPositionLoggerChannel(&channel)
		satellite.RunCartesianPositions()
	}
	return channels
}

func startSphericalPositionsSatellites(satellites SatelliteList) actors.SphericalPositionLoggerDeviceChannels {
	channels := make(actors.SphericalPositionLoggerDeviceChannels, 0)
	for _, satellite := range satellites {
		channel := make(actors.SphericalPositionLoggerDeviceChannel)
		channels = append(channels, &channel)
		satellite.SetSphericalPositionLoggerChannel(&channel)
		satellite.RunSphericalPositions()
	}
	return channels
}

func startDistancesSatellites(satellites SatelliteList) actors.DistanceLoggerDeviceChannels {
	channels := make(actors.DistanceLoggerDeviceChannels, 0)
	for _, satellite := range satellites {
		channel := make(actors.DistanceLoggerDeviceChannel)
		channels = append(channels, &channel)
		satellite.SetDistanceLoggerChannel(&channel)
		satellite.RunDistances()
	}
	return channels
}

func startSatellites(satellites SatelliteList) (actors.LoggerDeviceChannels,
	actors.LinkRequestChannels, actors.LinkRequestChannels,
	actors.ProgressTokenChannels, actors.AckTokenChannels, []string) {

	logChannels := make(actors.LoggerDeviceChannels, 0)
	tokenChannels := make(actors.ProgressTokenChannels, 0)
	ackChannels := make(actors.AckTokenChannels, 0)
	linkIncomingChannels := make(actors.LinkRequestChannels, 0)
	linkOutgoingChannels := make(actors.LinkRequestChannels, 0)
	satelliteNames := make([]string, 0)
	for _, satellite := range satellites {
		logChannel := make(actors.LoggerDeviceChannel)
		tokenChannel := make(actors.ProgressTokenChannel, 1)
		ackChannel := make(actors.AckTokenChannel, 1)
		linkIncomingChannel := make(actors.LinkRequestChannel, 5)
		linkOutgoingChannel := make(actors.LinkRequestChannel, 5)
		logChannels = append(logChannels, &logChannel)
		tokenChannels = append(tokenChannels, &tokenChannel)
		ackChannels = append(ackChannels, &ackChannel)
		linkIncomingChannels = append(linkIncomingChannels, &linkIncomingChannel)
		linkOutgoingChannels = append(linkOutgoingChannels, &linkOutgoingChannel)
		satelliteNames = append(satelliteNames, satellite.GetName())
		satellite.SetLoggerChannel(&logChannel)
		satellite.SetProgressTokenChannel(&tokenChannel)
		satellite.SetAckTokenChannel(&ackChannel)
		satellite.SetLinkerChannels(&linkIncomingChannel, &linkOutgoingChannel)
		satellite.Run()
	}
	return logChannels, linkIncomingChannels, linkOutgoingChannels, tokenChannels, ackChannels, satelliteNames
}
