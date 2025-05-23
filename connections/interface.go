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
	InterfaceId        int
	InterfaceOwner     string
	SendChannel        *chan Packet
	ReceiveChannel     *chan Packet
	Link               ILink
	DeviceConnectedTo  string
	LastPacketSentTime float64
}

type INetworkInterface interface {
	Send(packet Packet, timeOfEvent float64) (bool, int)
	Receive() []Event
	GetDeviceConnectedTo() string
	GetDeviceOwner() string
	GetLink() ILink
	ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet)
	CloseConnection()
}

type ILink interface {
	CalculateDeliveryTime(packet Packet) float64
	CalculateTransmissionTime(packet Packet) float64
	UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool
}

func (networkInterface *NetworkInterface) Send(packet Packet, timeOfEvent float64) (bool, int) {
	networkInterface.LastPacketSentTime = max(timeOfEvent, networkInterface.LastPacketSentTime)
	packet.PacketSentTime = networkInterface.LastPacketSentTime
	linkDown := networkInterface.Link.UpdateDistance(networkInterface.InterfaceOwner, networkInterface.DeviceConnectedTo, 0.001*networkInterface.LastPacketSentTime)

	if linkDown {
		networkInterface.CloseConnection()
		return false, int(packet.PacketSentTime)
	}

	if len(*networkInterface.SendChannel) >= cap(*networkInterface.SendChannel) {
		return false, int(packet.PacketSentTime)
	}

	transmissionTime := networkInterface.Link.CalculateTransmissionTime(packet)
	select {
	case *networkInterface.SendChannel <- packet:
		networkInterface.LastPacketSentTime += transmissionTime
	default:
		return false, int(packet.PacketSentTime)
	}

	return true, int(packet.PacketSentTime)
}

func (networkInterface *NetworkInterface) CloseConnection() {
	networkInterface.DeviceConnectedTo = ""
	close(*networkInterface.SendChannel)
	networkInterface.SendChannel = nil
	networkInterface.ReceiveChannel = nil
}

func (networkInterface *NetworkInterface) Receive() []Event {
	recievedEvents := make([]Event, 0)
	lastTimeStampRead := 0.0
	channelEmpty := false
	for !channelEmpty {
		select {
		case packet, ok := <-*networkInterface.ReceiveChannel:
			if !ok {
				channelEmpty = true
				networkInterface.CloseConnection()
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

func (networkInterface *NetworkInterface) ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet) {
	if networkInterface.DeviceConnectedTo == newDeviceConnectedTo {
		return
	}
	if networkInterface.DeviceConnectedTo != "" {
		networkInterface.CloseConnection()
	}
	networkInterface.DeviceConnectedTo = newDeviceConnectedTo
	networkInterface.SendChannel = newSendChannel
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
