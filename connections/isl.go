package connections

import (
	"math"

	"SatSimGo/helpers"
)

type ISL struct {
	SpeedOfLightVAC  float64
	Bitrate          float64
	PropagationDelay float64
	Bandwidth        float64
	LinkNoiseCoef    float64
	MaxPacketSize    float64
	GeoCalculation   helpers.IAnomalyCalculation
}

func (isl *ISL) UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool {
	ownerOrbit, ownerNum := helpers.GetOrbitAndSatelliteId(ownerId)
	connectedOrbit, connectedNum := helpers.GetOrbitAndSatelliteId(connectedId)
	updatedDistance := isl.GeoCalculation.CalculateDistanceBySatelliteId(ownerNum, ownerOrbit, connectedNum, connectedOrbit, float64(timeStamp))
	distanceKM := updatedDistance / 1000.0
	isl.PropagationDelay = 1000.0 * updatedDistance / isl.SpeedOfLightVAC
	isl.Bitrate = isl.Bandwidth * math.Log2(1+isl.LinkNoiseCoef/math.Pow(distanceKM, 2))
	return isl.isLinkOutOfRange(updatedDistance)
}

func (isl *ISL) isLinkOutOfRange(distance float64) bool {
	return distance > isl.GeoCalculation.GetMaxDistance()
}

func (isl *ISL) CalculateDeliveryTime(packet Packet) float64 {
	return isl.PropagationDelay + packet.Length/isl.Bitrate
}

// in ms
func (isl *ISL) CalculateTransmissionTime(packet Packet) float64 {
	return packet.Length / isl.Bitrate
}

func InitISLs(ownerSatellite string, numberOfIsls int, speedOfLightVAC float64, bandwidth float64, linkNoiseCoef float64,
	anomalyCalculations helpers.IAnomalyCalculation, maxPacketSize float64, interfaceBufferSize int) []INetworkInterface {
	islList := make([]INetworkInterface, numberOfIsls)
	for i := 0; i < numberOfIsls; i++ {
		islList[i] = &NetworkInterface{
			InterfaceId:         i,
			InterfaceOwner:      ownerSatellite,
			SendChannel:         nil,
			ReceiveChannel:      nil,
			Link:                &ISL{speedOfLightVAC, 0.0, 0.0, bandwidth, linkNoiseCoef, maxPacketSize, anomalyCalculations},
			DeviceConnectedTo:   "",
			BufferEndTimes:      make([]float64, 0),
			Buffer:              make([]Packet, 0),
			InterfaceBufferSize: interfaceBufferSize,
		}
	}
	return islList
}

func InitISL(ownerSatellite string, interfaceId int, speedOfLightVAC float64, bandwidth float64, linkNoiseCoef float64,
	anomalyCalculations helpers.IAnomalyCalculation, maxPacketSize float64, interfaceBufferSize int) INetworkInterface {
	return &NetworkInterface{
		InterfaceId:         interfaceId,
		InterfaceOwner:      ownerSatellite,
		SendChannel:         nil,
		ReceiveChannel:      nil,
		Link:                &ISL{speedOfLightVAC, 0.0, 0.0, bandwidth, linkNoiseCoef, maxPacketSize, anomalyCalculations},
		DeviceConnectedTo:   "",
		BufferEndTimes:      make([]float64, 0),
		Buffer:              make([]Packet, 0),
		InterfaceBufferSize: interfaceBufferSize,
	}
}

func (isl *ISL) Clone() ILink {
	return &ISL{
		SpeedOfLightVAC:  isl.SpeedOfLightVAC,
		Bitrate:          isl.Bitrate,
		PropagationDelay: isl.PropagationDelay,
		Bandwidth:        isl.Bandwidth,
		LinkNoiseCoef:    isl.LinkNoiseCoef,
		MaxPacketSize:    isl.MaxPacketSize,
		GeoCalculation:   isl.GeoCalculation,
	}
}
