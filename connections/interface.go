package connections

type ILink interface {
	calculateDeliveryTime(packet Packet) int
	updateLink(distance float64)
}

type NetworkInterface struct {
	InterfaceId        int
	SendChannel        chan Packet
	ReceiveChannel     chan Packet
	Link               ILink
	DeviceConnectedTo  string
	LastPacketSentTime int
}

type INetworkInterface interface {
	Send(packet Packet)
	Receive() Packet
	GetDeviceConnectedTo() string
	GetLink() ILink
	changeLink(newLink ILink)
}

func (networkInterface *NetworkInterface) Send(packet Packet) {
	networkInterface.SendChannel <- packet
}

func (networkInterface *NetworkInterface) Receive() Packet {
	packet := <-networkInterface.ReceiveChannel
	packet.PacketDeliveryTime = packet.PacketDeliveryTime + networkInterface.Link.calculateDeliveryTime(packet)
	return packet
}

func (networkInterface *NetworkInterface) GetDeviceConnectedTo() string {
	return networkInterface.DeviceConnectedTo
}

func (networkInterface *NetworkInterface) ChangeLink(newLink ILink) {
	networkInterface.Link = newLink
}

func (networkInterface *NetworkInterface) GetLink() ILink {
	return networkInterface.Link
}
