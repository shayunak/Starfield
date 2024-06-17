package connections

type Packet struct {
	Source             string
	Destination        string
	Length             int
	PacketDeliveryTime int
}
