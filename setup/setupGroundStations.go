package setup

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/shayunak/SatSimGo/actors"
	"github.com/shayunak/SatSimGo/helpers"
)

type GroundStationList []actors.IGroundStation

func calculateElevationLimitRatio(earthRadius float64, orbitRadius float64, minElevationAngle float64, altitude float64) float64 {
	minElevationAngleTangent := math.Tan(minElevationAngle * math.Pi / 180.0)
	earthOrbitRatio := earthRadius / orbitRadius
	altitudeOrbitRatio := altitude / orbitRadius

	elevationTerm := minElevationAngleTangent * math.Sqrt(math.Pow(minElevationAngleTangent, 2.0)+altitudeOrbitRatio*(2.0-altitudeOrbitRatio))

	return (earthOrbitRatio + elevationTerm) / (1.0 + math.Pow(minElevationAngleTangent, 2.0))
}

func openGroundStationFiles(fileName string) (*os.File, *csv.Reader) {
	groundStationsFilePath := "./configs/" + fileName
	file, err := os.Open(groundStationsFilePath)
	if err != nil {
		panic(err)
	}

	csvReader := csv.NewReader(file)

	_, err = csvReader.Read()
	if err != nil {
		panic(err)
	}

	return file, csvReader
}

func initGroundStations(groundStations *GroundStationList, config Config, groundStationFileName string,
	groundCalc helpers.IGroundStationCalculation, timeStep int, totalSimulationTime int) {
	totalSimulationTimeMilliseconds := totalSimulationTime * 1000
	groundStationFile, groundStationCoordinates := openGroundStationFiles(groundStationFileName)
	groundStationSpecs := make(helpers.GroundStationSpecs)

	for {
		record, err := groundStationCoordinates.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error reading CSV data:", err)
			break
		}
		groundStationName := record[0]
		latitude, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			fmt.Println("Cannot convert latitude of ground station ", record[1], ":", err)
			continue
		}
		longitude, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			fmt.Println("Cannot convert longitude of ground station ", record[2], ":", err)
			continue
		}
		groundStationLatitude := latitude * math.Pi / 180.0
		groundStationLongitude := longitude * math.Pi / 180.0
		ascension, anomaly := groundCalc.FindCoordinatesOfTheAboveHeadPoint(groundStationName, groundStationLatitude, groundStationLongitude)
		anomalyEl := helpers.AnomalyElements{
			AnomalySinus:   math.Sin(anomaly),
			AnomalyCosinus: math.Cos(anomaly),
		}
		groundStationSpecs[groundStationName] = helpers.GroundStationSpec{
			Latitude:           groundStationLatitude,
			Longitude:          groundStationLongitude,
			HeadPointAscension: ascension,
			HeadPointAnomalyEl: anomalyEl,
		}
		*groundStations = append(*groundStations,
			actors.NewGroundStation(groundStationName, groundStationLatitude, groundStationLongitude, timeStep,
				totalSimulationTimeMilliseconds, anomaly, ascension, groundCalc, config.SatelliteConfig.SpeedOfLightVac,
				config.SatelliteConfig.GSLBandwidth, config.SatelliteConfig.GSLLinkNoiseCoef, anomalyEl,
				config.SatelliteConfig.MaxPacketSize, config.SatelliteConfig.InterfaceBufferSize),
		)
	}
	groundStationFile.Close()

	groundCalc.SetGroundStationSpecs(&groundStationSpecs)
}

func startGroundStations(groundStations GroundStationList) (actors.SpaceDeviceChannels, []string) {
	channels := make(actors.SpaceDeviceChannels, 0)
	groundStationNames := make([]string, 0)
	for _, groundStation := range groundStations {
		channel := make(actors.SpaceDeviceChannel)
		channels = append(channels, &channel)
		groundStationNames = append(groundStationNames, groundStation.GetName())
		groundStation.SetSpaceChannel(&channel)
		groundStation.Run()
	}
	return channels, groundStationNames
}

func startDistancesGroundStations(groundStations GroundStationList) actors.DistanceSpaceDeviceChannels {
	channels := make(actors.DistanceSpaceDeviceChannels, 0)
	for _, groundStation := range groundStations {
		channel := make(actors.DistanceSpaceDeviceChannel)
		channels = append(channels, &channel)
		groundStation.SetDistanceSpaceChannel(&channel)
		groundStation.RunDistances()
	}
	return channels
}
