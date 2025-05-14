package connections

import (
	"math"

	"github.com/shayunak/SatSimGo/helpers"
)

type ISL struct {
	SpeedOfLightVAC  float64
	Bitrate          float64
	PropagationDelay float64
	Bandwidth        float64
	LinkNoiseCoef    float64
	GeoCalculation   helpers.IAnomalyCalculation
}

func (isl *ISL) UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool {
	ownerOrbit, ownerNum := helpers.GetOrbitAndSatelliteId(ownerId)
	connectedOrbit, connectedNum := helpers.GetOrbitAndSatelliteId(connectedId)
	updatedDistance := isl.GeoCalculation.CalculateDistanceBySatelliteId(ownerNum, ownerOrbit, connectedNum, connectedOrbit, float64(timeStamp))
	distanceKM := updatedDistance / 1000.0
	isl.PropagationDelay = updatedDistance / isl.SpeedOfLightVAC
	isl.Bitrate = isl.Bandwidth * math.Log2(1+isl.LinkNoiseCoef/math.Pow(distanceKM, 2))
	return isl.isLinkOutOfRange(updatedDistance)
}

func (isl *ISL) isLinkOutOfRange(distance float64) bool {
	if distance > isl.GeoCalculation.GetMaxDistance() {
		return true
	}
	return false
}

func (isl *ISL) CalculateDeliveryTime(packet Packet) float64 {
	return isl.PropagationDelay + float64(packet.Length)/isl.Bitrate
}

// in ms
func (isl *ISL) CalculateTransmissionTime(packet Packet) float64 {
	return packet.Length / isl.Bitrate
}

func InitISLs(ownerSatellite string, numberOfIsls int, speedOfLightVAC float64, bandwidth float64, linkNoiseCoef float64,
	anomalyCalculations helpers.IAnomalyCalculation) []INetworkInterface {
	islList := make([]INetworkInterface, numberOfIsls)
	for i := 0; i < numberOfIsls; i++ {
		islList[i] = &NetworkInterface{
			InterfaceId:        i,
			InterfaceOwner:     ownerSatellite,
			IsLinkDown:         false,
			SendChannel:        nil,
			ReceiveChannel:     nil,
			Link:               &ISL{speedOfLightVAC, 0.0, 0.0, bandwidth, linkNoiseCoef, anomalyCalculations},
			DeviceConnectedTo:  "",
			LastPacketSentTime: 0,
		}
	}
	return islList
}
