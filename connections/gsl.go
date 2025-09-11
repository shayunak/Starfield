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
	MaxPacketSize    float64
	GeoCalculation   helpers.IGroundStationCalculation
	GeometricSpec    *GeoSpec
}

func (gsl *GSL) UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool {
	var updatedDistance float64
	var isLinkInRange bool
	if gsl.GeometricSpec.Orbit != nil {
		updatedDistance, isLinkInRange = gsl.GeoCalculation.CalculateCoveringGSDistance(connectedId, timeStamp, gsl.GeometricSpec.Anomaly, gsl.GeometricSpec.Orbit)
		if !isLinkInRange {
			return true
		}
	} else {
		updatedDistance, isLinkInRange = gsl.GeoCalculation.FindSatellite(connectedId, gsl.GeometricSpec.HeadPointAnomalyEl, gsl.GeometricSpec.HeadPointAscension, timeStamp)
		if !isLinkInRange {
			return true
		}
	}
	distanceKM := updatedDistance / 1000.0
	gsl.PropagationDelay = 1000.0 * updatedDistance / gsl.SpeedOfLightVAC
	gsl.Bitrate = gsl.Bandwidth * math.Log2(1+gsl.LinkNoiseCoef/math.Pow(distanceKM, 2))
	return false
}

func (gsl *GSL) CalculateDeliveryTime(packet Packet) float64 {
	return gsl.PropagationDelay + packet.Length/gsl.Bitrate
}

func (gsl *GSL) CalculateTransmissionTime(packet Packet) float64 {
	return packet.Length / gsl.Bitrate
}

func InitGSL(owner string, speedOfLightVAC float64, bandwidth float64, linkNoiseCoef float64,
	orbit helpers.IOrbit, anomaly float64, headPointAscension float64, headPointAnomalyEl helpers.AnomalyElements,
	groundStationCalculations helpers.IGroundStationCalculation, maxPacketSize float64, interfaceBufferSize int) INetworkInterface {
	return &NetworkInterface{
		InterfaceId:    0,
		InterfaceOwner: owner,
		SendChannel:    nil,
		ReceiveChannel: nil,
		Link: &GSL{
			SpeedOfLightVAC:  speedOfLightVAC,
			Bitrate:          0.0,
			PropagationDelay: 0.0,
			Bandwidth:        bandwidth,
			LinkNoiseCoef:    linkNoiseCoef,
			GeoCalculation:   groundStationCalculations,
			MaxPacketSize:    maxPacketSize,
			GeometricSpec: &GeoSpec{
				Anomaly:            anomaly,
				Orbit:              orbit,
				HeadPointAscension: headPointAscension,
				HeadPointAnomalyEl: headPointAnomalyEl,
			},
		},
		DeviceConnectedTo:   "",
		BufferEndTimes:      make([]float64, 0),
		Buffer:              make([]Packet, 0),
		InterfaceBufferSize: interfaceBufferSize,
	}
}

func (gsl *GSL) Clone() ILink {
	return &GSL{
		SpeedOfLightVAC:  gsl.SpeedOfLightVAC,
		Bitrate:          gsl.Bitrate,
		PropagationDelay: gsl.PropagationDelay,
		Bandwidth:        gsl.Bandwidth,
		LinkNoiseCoef:    gsl.LinkNoiseCoef,
		MaxPacketSize:    gsl.MaxPacketSize,
		GeoCalculation:   gsl.GeoCalculation,
		GeometricSpec:    gsl.GeometricSpec,
	}
}
