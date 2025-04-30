package connections

import (
	"github.com/shayunak/SatSimGo/helpers"
)

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
	IsLinkDown         bool
	SendChannel        *chan Packet
	ReceiveChannel     *chan Packet
	Link               ILink
	DeviceConnectedTo  string
	LastPacketSentTime float64
	GeoCalculation     helpers.IAnomalyCalculation
}

type INetworkInterface interface {
	Send(packet Packet, timeOfEvent float64) (bool, int)
	Receive() []Event
	GetDeviceConnectedTo() string
	GetLink() ILink
	ChangeLink(newDeviceConnectedTo string, newSendChannel *chan Packet, newReceiveChannel *chan Packet)
	CloseConnection()
}

type ILink interface {
	CalculateDeliveryTime(packet Packet) float64
	CalculateTransmissionTime(packet Packet) float64
	UpdateLink(distance float64)
}

func (networkInterface *NetworkInterface) updateLink(timeStamp float64) {
	ownerOrbit, ownerId := helpers.GetOrbitAndSatelliteId(networkInterface.InterfaceOwner)
	connectedOrbit, connectedId := helpers.GetOrbitAndSatelliteId(networkInterface.DeviceConnectedTo)
	updatedDistance := networkInterface.GeoCalculation.CalculateDistanceBySatelliteId(ownerId, ownerOrbit, connectedId, connectedOrbit, float64(timeStamp))
	if updatedDistance > networkInterface.GeoCalculation.GetMaxDistance() {
		networkInterface.IsLinkDown = true
	} else {
		networkInterface.IsLinkDown = false
	}
	networkInterface.Link.UpdateLink(updatedDistance)
}

func (networkInterface *NetworkInterface) Send(packet Packet, timeOfEvent float64) (bool, int) {
	networkInterface.LastPacketSentTime = max(timeOfEvent, networkInterface.LastPacketSentTime)
	packet.PacketSentTime = networkInterface.LastPacketSentTime
	networkInterface.updateLink(0.001 * networkInterface.LastPacketSentTime)

	if networkInterface.IsLinkDown || len(*networkInterface.SendChannel) >= cap(*networkInterface.SendChannel) {
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
			networkInterface.updateLink(0.001 * packet.PacketSentTime)
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
					InterfaceId:     pair.Id,
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
			InterfaceId:     entry.InterfaceId,
			ConnectedDevice: entry.ConnectedDevice,
			SendChannel:     entry.SendChannel,
			ReceiveChannel:  channel,
		}
	}

	return topologyList
}
