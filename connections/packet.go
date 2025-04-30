package connections

type Packet struct {
	PacketId       int
	Source         string
	Destination    string
	Length         int
	PacketSentTime float64
}
