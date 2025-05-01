package connections

type Packet struct {
	PacketId       int
	Source         string
	Destination    string
	Length         float64
	PacketSentTime float64
}
