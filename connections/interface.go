package connections

type ILink interface {
	CalculateDeliveryTime(packet Packet) int
	UpdateLink(distance float64)
}

type NetworkInterface struct {
	InterfaceId        int
	SendChannel        *chan Packet
	ReceiveChannel     *chan Packet
	Link               ILink
	DeviceConnectedTo  string
	LastPacketSentTime int
}

type INetworkInterface interface {
	Send(packet Packet)
	Receive() Packet
	GetDeviceConnectedTo() string
	GetLink() ILink
	ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet)
}

func (networkInterface *NetworkInterface) Send(packet Packet) {
	*networkInterface.SendChannel <- packet
}

func (networkInterface *NetworkInterface) Receive() Packet {
	packet := <-*networkInterface.ReceiveChannel
	packet.PacketSentTime = packet.PacketSentTime + networkInterface.Link.CalculateDeliveryTime(packet)
	return packet
}

func (networkInterface *NetworkInterface) GetDeviceConnectedTo() string {
	return networkInterface.DeviceConnectedTo
}

func (networkInterface *NetworkInterface) ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet) {
	networkInterface.DeviceConnectedTo = newDeviceConnectedTo
	networkInterface.SendChannel = newSendChannel
	networkInterface.ReceiveChannel = newReceiveChannel
}

func (networkInterface *NetworkInterface) GetLink() ILink {
	return networkInterface.Link
}
