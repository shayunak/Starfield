package connections

import "math"

type ISL struct {
	SpeedOfLightVAC  float64
	Bitrate          float64
	PropagationDelay float64
	Bandwidth        float64
	LinkNoiseCoef    float64
}

func (isl *ISL) calculateDeliveryTime(packet Packet) int {
	return int(isl.PropagationDelay + float64(packet.Length)/isl.Bitrate)
}

func (isl *ISL) updateLink(distance float64) {
	distanceKM := distance / 1000.0
	isl.PropagationDelay = distance / isl.SpeedOfLightVAC
	isl.Bitrate = isl.Bandwidth * math.Log2(1+isl.LinkNoiseCoef/math.Pow(distanceKM, 2))
}
