package connections

type Pair struct {
	Id              int    // pair id
	FirstSatellite  string // sending satellite
	SecondSatellite string // receiving satellite
}

type InterfaceEntry struct {
	InterfaceId     int
	ConnectedDevice string
	SendChannel     *chan Packet
	ReceiveChannel  *chan Packet
}

const SEND_EVENT int = 0
const RECEIVE_EVENT int = 1

type NetworkInterface struct {
	/* Chance to buffer packets if the link is down*/
	InterfaceId         int
	InterfaceOwner      string
	SendChannel         *chan Packet
	ReceiveChannel      *chan Packet
	Link                ILink
	DeviceConnectedTo   string
	BufferEndTimes      []float64
	InterfaceBufferSize int
	Buffer              []Packet
}

type INetworkInterface interface {
	IsBufferEmpty() bool
	Send(packet Packet, timeOfEvent float64) (bool, int)
	Receive() []Event
	HasReceiveChannel() bool
	HasSendChannel() bool
	GetDeviceConnectedTo() string
	GetDeviceOwner() string
	GetLink() ILink
	ProcessBuffer()
	ChangeSendLink(newDeviceConnectedTo string, newSendChannel *chan Packet)
	ChangeReceiveLink(newDeviceConnectedTo string, newReceiveChannel *chan Packet)
	CloseSendSideConnection()
	CloseReceiveSideConnection()
	Clone() INetworkInterface
	shouldDropPacket(packetTime float64, transmissionTime float64) bool
	getLastPacketEndTime() float64
}

type ILink interface {
	CalculateDeliveryTime(packet Packet) float64
	CalculateTransmissionTime(packet Packet) float64
	UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool
	Clone() ILink
}

func (networkInterface *NetworkInterface) IsBufferEmpty() bool {
	return len(networkInterface.Buffer) == 0
}

func (networkInterface *NetworkInterface) HasReceiveChannel() bool {
	return networkInterface.ReceiveChannel != nil
}

func (networkInterface *NetworkInterface) HasSendChannel() bool {
	return networkInterface.SendChannel != nil
}

func (networkInterface *NetworkInterface) Clone() INetworkInterface {
	return &NetworkInterface{
		InterfaceId:         networkInterface.InterfaceId,
		InterfaceOwner:      networkInterface.InterfaceOwner,
		SendChannel:         networkInterface.SendChannel,
		ReceiveChannel:      networkInterface.ReceiveChannel,
		Link:                networkInterface.Link.Clone(),
		DeviceConnectedTo:   networkInterface.DeviceConnectedTo,
		BufferEndTimes:      make([]float64, 0),
		InterfaceBufferSize: networkInterface.InterfaceBufferSize,
		Buffer:              make([]Packet, 0),
	}
}

func (networkInterface *NetworkInterface) getLastPacketEndTime() float64 {
	if len(networkInterface.BufferEndTimes) > 0 {
		return networkInterface.BufferEndTimes[len(networkInterface.BufferEndTimes)-1]
	}
	return 0.0
}

func (networkInterface *NetworkInterface) shouldDropPacket(timeOfEvent float64, packetEndTime float64) bool {
	i := 0
	for i < len(networkInterface.BufferEndTimes) && timeOfEvent > networkInterface.BufferEndTimes[i] {
		i++
	}
	if i == 0 && len(networkInterface.BufferEndTimes) >= networkInterface.InterfaceBufferSize {
		return true
	}
	if i >= len(networkInterface.BufferEndTimes) {
		networkInterface.BufferEndTimes = []float64{packetEndTime}
	} else {
		networkInterface.BufferEndTimes = append(networkInterface.BufferEndTimes[i+1:], packetEndTime)
	}
	return false
}

func (networkInterface *NetworkInterface) ProcessBuffer() {
	i := 0
	for i < len(networkInterface.Buffer) && len(*networkInterface.SendChannel) < cap(*networkInterface.SendChannel) {
		*networkInterface.SendChannel <- networkInterface.Buffer[i]
		i++
	}
	networkInterface.Buffer = networkInterface.Buffer[i:]
}

func (networkInterface *NetworkInterface) Send(packet Packet, timeOfEvent float64) (bool, int) {
	lastPacketEndTime := networkInterface.getLastPacketEndTime()
	packetSentTime := max(timeOfEvent, lastPacketEndTime)
	linkDown := networkInterface.Link.UpdateDistance(networkInterface.InterfaceOwner, networkInterface.DeviceConnectedTo, 0.001*packetSentTime)

	if linkDown {
		//networkInterface.CloseSendSideConnection() We don't know, no packet order
		return true, int(packetSentTime) // packet dropped
	}

	packetEndTime := packetSentTime + networkInterface.Link.CalculateTransmissionTime(packet)
	if networkInterface.shouldDropPacket(timeOfEvent, packetEndTime) {
		return true, int(packetSentTime) // packet dropped
	}

	packet.PacketSentTime = packetSentTime
	if len(*networkInterface.SendChannel) < cap(*networkInterface.SendChannel) {
		if len(networkInterface.Buffer) > 0 {
			networkInterface.Buffer = append(networkInterface.Buffer, packet)
			networkInterface.ProcessBuffer()
		} else {
			*networkInterface.SendChannel <- packet
		}
	} else {
		networkInterface.Buffer = append(networkInterface.Buffer, packet)
	}

	return false, int(packet.PacketSentTime) // packet sent
}

func (networkInterface *NetworkInterface) CloseSendSideConnection() {
	close(*networkInterface.SendChannel)
	networkInterface.SendChannel = nil
	if networkInterface.ReceiveChannel == nil {
		networkInterface.DeviceConnectedTo = ""
	}
}

func (networkInterface *NetworkInterface) CloseReceiveSideConnection() {
	networkInterface.ReceiveChannel = nil
	if networkInterface.SendChannel == nil {
		networkInterface.DeviceConnectedTo = ""
	}
}

func (networkInterface *NetworkInterface) Receive() []Event {
	recievedEvents := make([]Event, 0)
	lastTimeStampRead := 0.0
	channelEmpty := false
	for !channelEmpty {
		select {
		case packet, ok := <-*networkInterface.ReceiveChannel:
			if !ok && (len(*networkInterface.ReceiveChannel) == 0) {
				channelEmpty = true
				//networkInterface.CloseReceiveSideConnection() no packet order
				break
			}
			networkInterface.Link.UpdateDistance(networkInterface.InterfaceOwner, networkInterface.DeviceConnectedTo, 0.001*packet.PacketSentTime)
			lastTimeStampRead = packet.PacketSentTime + networkInterface.Link.CalculateDeliveryTime(packet)
			event := Event{
				TimeStamp: lastTimeStampRead,
				Type:      SEND_EVENT,
				Data:      &packet,
			}
			recievedEvents = append(recievedEvents, event)
		default:
			channelEmpty = true
		}
	}

	return recievedEvents
}

func (networkInterface *NetworkInterface) GetDeviceConnectedTo() string {
	return networkInterface.DeviceConnectedTo
}

func (networkInterface *NetworkInterface) GetDeviceOwner() string {
	return networkInterface.InterfaceOwner
}

func (networkInterface *NetworkInterface) ChangeSendLink(newDeviceConnectedTo string, newSendChannel *chan Packet) {
	networkInterface.DeviceConnectedTo = newDeviceConnectedTo
	networkInterface.SendChannel = newSendChannel
}

func (networkInterface *NetworkInterface) ChangeReceiveLink(newDeviceConnectedTo string, newReceiveChannel *chan Packet) {
	networkInterface.DeviceConnectedTo = newDeviceConnectedTo
	networkInterface.ReceiveChannel = newReceiveChannel
}

func (networkInterface *NetworkInterface) GetLink() ILink {
	return networkInterface.Link
}

func GetTopologyList(pairs []Pair, interfaceBufferSize int) map[string]map[string]InterfaceEntry {
	topologyList := make(map[string]map[string]InterfaceEntry)

	// First pass initializing the matrix
	for _, pair := range pairs {
		if topologyList[pair.FirstSatellite] == nil {
			topologyList[pair.FirstSatellite] = make(map[string]InterfaceEntry)
		}
		channel := make(chan Packet, interfaceBufferSize)
		topologyList[pair.FirstSatellite][pair.SecondSatellite] = InterfaceEntry{
			InterfaceId:     pair.Id,
			ConnectedDevice: pair.SecondSatellite,
			SendChannel:     &channel,
			ReceiveChannel:  nil,
		}
	}

	// Second pass assigning recieve channels
	for _, pair := range pairs {
		entry := topologyList[pair.FirstSatellite][pair.SecondSatellite]
		entry.ReceiveChannel = topologyList[pair.SecondSatellite][pair.FirstSatellite].SendChannel
		topologyList[pair.FirstSatellite][pair.SecondSatellite] = entry
	}

	return topologyList
}
