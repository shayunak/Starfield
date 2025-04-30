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
}

func (isl *ISL) CalculateDeliveryTime(packet Packet) float64 {
	return isl.PropagationDelay + float64(packet.Length)/isl.Bitrate
}

func (isl *ISL) UpdateLink(distance float64) {
	distanceKM := distance / 1000.0
	isl.PropagationDelay = distance / isl.SpeedOfLightVAC
	isl.Bitrate = isl.Bandwidth * math.Log2(1+isl.LinkNoiseCoef/math.Pow(distanceKM, 2))
}

func (isl *ISL) CalculateTransmissionTime(packet Packet) float64 {
	return float64(packet.Length) / isl.Bitrate
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
			Link:               &ISL{speedOfLightVAC, 0.0, 0.0, bandwidth, linkNoiseCoef},
			DeviceConnectedTo:  "",
			LastPacketSentTime: 0,
			GeoCalculation:     anomalyCalculations,
		}
	}
	return islList
}
