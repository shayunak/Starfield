package connections

import (
	"math"

	"github.com/shayunak/SatSimGo/helpers"
)

type GeoSpec struct {
	Anomaly            float64
	Orbit              helpers.IOrbit
	HeadPointAscension float64
	HeadPointAnomalyEl helpers.AnomalyElements
}

type GSL struct {
	SpeedOfLightVAC  float64
	Bitrate          float64
	PropagationDelay float64
	Bandwidth        float64
	LinkNoiseCoef    float64
	GeoCalculation   helpers.IGroundStationCalculation
	GeometricSpec    *GeoSpec
}

func (gsl *GSL) UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool {
	var updatedDistance float64
	var isLinkInRange bool
	if gsl.GeometricSpec.Orbit != nil {
		updatedAnomaly, _ := gsl.GeoCalculation.GetAnomalyCalculations().UpdatePosition(gsl.GeometricSpec.Anomaly, timeStamp)
		updatedDistance, isLinkInRange = gsl.GeoCalculation.GetCoveringGroundStations(timeStamp, updatedAnomaly, gsl.GeometricSpec.Orbit)[connectedId]
		if !isLinkInRange {
			return false
		}
	} else {
		updatedAscension := gsl.GeoCalculation.UpdatePosition(gsl.GeometricSpec.HeadPointAscension, timeStamp)
		updatedDistance, isLinkInRange = gsl.GeoCalculation.FindSatellitesInRange(ownerId, updatedAscension, gsl.GeometricSpec.HeadPointAnomalyEl, timeStamp)[connectedId]
		if !isLinkInRange {
			return false
		}
	}
	distanceKM := updatedDistance / 1000.0
	gsl.PropagationDelay = updatedDistance / gsl.SpeedOfLightVAC
	gsl.Bitrate = gsl.Bandwidth * math.Log2(1+gsl.LinkNoiseCoef/math.Pow(distanceKM, 2))
	return true
}

func (gsl *GSL) CalculateDeliveryTime(packet Packet) float64 {
	return gsl.PropagationDelay + float64(packet.Length)/gsl.Bitrate
}

func (gsl *GSL) CalculateTransmissionTime(packet Packet) float64 {
	return packet.Length / gsl.Bitrate
}

func InitGSL(owner string, speedOfLightVAC float64, bandwidth float64, linkNoiseCoef float64,
	orbit helpers.IOrbit, anomaly float64, headPointAscension float64, headPointAnomalyEl helpers.AnomalyElements,
	groundStationCalculations helpers.IGroundStationCalculation) INetworkInterface {
	return &NetworkInterface{
		InterfaceId:    0,
		InterfaceOwner: owner,
		IsLinkDown:     false,
		SendChannel:    nil,
		ReceiveChannel: nil,
		Link: &GSL{
			SpeedOfLightVAC:  speedOfLightVAC,
			Bitrate:          0.0,
			PropagationDelay: 0.0,
			Bandwidth:        bandwidth,
			LinkNoiseCoef:    linkNoiseCoef,
			GeoCalculation:   groundStationCalculations,
			GeometricSpec: &GeoSpec{
				Anomaly:            anomaly,
				Orbit:              orbit,
				HeadPointAscension: headPointAscension,
				HeadPointAnomalyEl: headPointAnomalyEl,
			},
		},
		DeviceConnectedTo:  "",
		LastPacketSentTime: 0,
	}
}
