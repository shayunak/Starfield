package connections

type Pair struct {
	FirstSatellite  string // sending satellite
	SecondSatellite string // receiving satellite
}

type InterfaceEntry struct {
	ConnectedDevice string
	SendChannel     *chan Packet
	ReceiveChannel  *chan Packet
}

type Event struct {
	TimeStamp int
	Type      int
	Data      *Packet
}

const SEND_EVENT int = 0
const RECEIVE_EVENT int = 1

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
	Receive(maxTimeStamp int) []Event
	GetDeviceConnectedTo() string
	GetLink() ILink
	ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet)
	CloseConnection()
}

type ILink interface {
	CalculateDeliveryTime(packet Packet) int
	UpdateLink(distance float64)
}

func (networkInterface *NetworkInterface) Send(packet Packet) {
	*networkInterface.SendChannel <- packet
}

func (networkInterface *NetworkInterface) CloseConnection() {
	networkInterface.DeviceConnectedTo = ""
	close(*networkInterface.SendChannel)
	networkInterface.SendChannel = nil
	networkInterface.ReceiveChannel = nil
}

func (networkInterface *NetworkInterface) Receive(maxTimeStamp int) []Event {
	recievedEvents := make([]Event, 0)
	lastTimeStampRead := 0
	channelEmpty := false
	for lastTimeStampRead <= maxTimeStamp && !channelEmpty {
		select {
		case packet, ok := <-*networkInterface.ReceiveChannel:
			if !ok {
				channelEmpty = true
				networkInterface.CloseConnection()
				break
			}
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

func (networkInterface *NetworkInterface) ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet) {
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
			if topologyList[pair.SecondSatellite] == nil {
				channel := make(chan Packet, interfaceBufferSize)
				topologyList[pair.FirstSatellite][pair.SecondSatellite] = InterfaceEntry{
					ConnectedDevice: pair.SecondSatellite,
					SendChannel:     &channel,
					ReceiveChannel:  nil,
				}
			}
		}
	}

	// Second pass assigning recieve channels
	for _, pair := range pairs {
		entry := topologyList[pair.SecondSatellite][pair.FirstSatellite]
		channel := topologyList[pair.FirstSatellite][pair.SecondSatellite].SendChannel
		topologyList[pair.SecondSatellite][pair.FirstSatellite] = InterfaceEntry{
			ConnectedDevice: entry.ConnectedDevice,
			SendChannel:     entry.SendChannel,
			ReceiveChannel:  channel,
		}
	}

	return topologyList
}
