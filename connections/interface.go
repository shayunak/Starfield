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
	Send(packet Packet, timeOfEvent float64) (bool, bool, int)
	Receive() []Event
	HasReceiveChannel() bool
	HasSendChannel() bool
	GetDeviceConnectedTo() string
	GetDeviceOwner() string
	GetLink() ILink
	ChangeSendLink(newDeviceConnectedTo string, newSendChannel *chan Packet)
	ChangeReceiveLink(newDeviceConnectedTo string, newReceiveChannel *chan Packet)
	CloseSendSideConnection()
	CloseReceiveSideConnection()
	Clone() INetworkInterface
}

type ILink interface {
	CalculateDeliveryTime(packet Packet) float64
	CalculateTransmissionTime(packet Packet) float64
	UpdateDistance(ownerId string, connectedId string, timeStamp float64) bool
	calculateBufferThresholdTime() float64
	Clone() ILink
}

func (networkInterface *NetworkInterface) HasReceiveChannel() bool {
	return networkInterface.ReceiveChannel != nil
}

func (networkInterface *NetworkInterface) HasSendChannel() bool {
	return networkInterface.SendChannel != nil
}

func (networkInterface *NetworkInterface) Clone() INetworkInterface {
	return &NetworkInterface{
		InterfaceId:        networkInterface.InterfaceId,
		InterfaceOwner:     networkInterface.InterfaceOwner,
		SendChannel:        networkInterface.SendChannel,
		ReceiveChannel:     networkInterface.ReceiveChannel,
		Link:               networkInterface.Link.Clone(),
		DeviceConnectedTo:  networkInterface.DeviceConnectedTo,
		LastPacketSentTime: networkInterface.LastPacketSentTime,
	}
}

func (networkInterface *NetworkInterface) Send(packet Packet, timeOfEvent float64) (bool, bool, int) {
	if len(*networkInterface.SendChannel) >= cap(*networkInterface.SendChannel) {
		deltaTime := timeOfEvent - networkInterface.LastPacketSentTime
		thresholdTime := networkInterface.Link.calculateBufferThresholdTime()
		if deltaTime > thresholdTime {
			return false, true, int(packet.PacketSentTime) // packet can be buffered
		} else {
			return true, false, int(packet.PacketSentTime) // packet dropped
		}
	}

	linkDown := networkInterface.Link.UpdateDistance(networkInterface.InterfaceOwner, networkInterface.DeviceConnectedTo, 0.001*networkInterface.LastPacketSentTime)

	if linkDown {
		networkInterface.CloseSendSideConnection()
		return true, false, int(packet.PacketSentTime) // packet dropped
	}

	networkInterface.LastPacketSentTime = max(timeOfEvent, networkInterface.LastPacketSentTime)
	packet.PacketSentTime = networkInterface.LastPacketSentTime
	transmissionTime := networkInterface.Link.CalculateTransmissionTime(packet)
	select {
	case *networkInterface.SendChannel <- packet:
		networkInterface.LastPacketSentTime += transmissionTime
	default:
		return true, false, int(packet.PacketSentTime) // packet dropped
	}

	return false, false, int(packet.PacketSentTime) // packet sent
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
				networkInterface.CloseReceiveSideConnection()
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
