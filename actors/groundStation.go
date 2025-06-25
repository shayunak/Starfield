package actors

import (
	"container/heap"
	"log"
	"math"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
)

type TrafficEntry struct {
	Destination string
	TimeStamp   int     // in milliseconds
	Length      float64 // in Mb
}

type GroundStation struct {
	// General
	Name                string
	Dt                  int // in milliseconds
	TimeStamp           int // in milliseconds
	TotalSimulationTime int // in milliseconds

	// Geometrical parameters
	Longitude          float64
	Latitude           float64
	HeadPointAnomaly   float64
	HeadPointAscension float64
	HeadPointAnomalyEl helpers.AnomalyElements
	GSCalculation      helpers.IGroundStationCalculation

	// Packet Level Simulation
	ForwardingTable map[int]ForwardingEntry
	EventQueue      connections.PriorityQueue

	// Goroutines and connections, and channels
	InterfaceBufferSize   int
	GSLInterfaceSample    connections.INetworkInterface
	GSLInterfaces         map[string]connections.INetworkInterface
	DistanceLoggerChannel *DistanceLoggerDeviceChannel
	LoggerChannel         *LoggerDeviceChannel
	LinkerOutgoingChannel *LinkRequestChannel
	LinkerIncomingChannel *LinkRequestChannel
	PendingConnections    []LinkRequest
}

type IGroundStation interface {
	getTimeStamp() int
	getTotalSimulationTime() int
	GetName() string
	// Distance Mode
	RunDistances()
	nextTimeStep()
	updatePosition()
	logDistances()
	SetDistanceLoggerChannel(channel *DistanceLoggerDeviceChannel)
	GetDistanceLoggerChannel() *DistanceLoggerDeviceChannel
	// Simulation Mode
	GetNumberOfPackets() int
	Run()
	SetLoggerChannel(channel *LoggerDeviceChannel)
	SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	GenerateTraffic(fromId int, traffic []TrafficEntry, maxPacketSize float64) (int, int)
	ReceiveFromInterfaces()
	SendPackets()
	ProcessBuffers()
	CheckIncomingConnection()
	SendPendingRequests()
	generatePackets(fromId int, maxPacketSize float64, entry TrafficEntry) ([]connections.Packet, int)
	logEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string)
	findConnection(toSatellite string) connections.INetworkInterface
	establishConnection(toSatellite string) connections.INetworkInterface
	establishSendChannel(inface connections.INetworkInterface, toSatellite string)
}

func (gs *GroundStation) GetName() string {
	return gs.Name
}

func (gs *GroundStation) getTimeStamp() int {
	return gs.TimeStamp
}

func (gs *GroundStation) getTotalSimulationTime() int {
	return gs.TotalSimulationTime
}

func NewGroundStation(name string, latitude float64, longitude float64, dt int, totalSimulationTime int,
	headPointAnomaly float64, headPointAscension float64, groundStationCalculation helpers.IGroundStationCalculation,
	speedOfLightVac float64, bandwidth float64, linkNoiseCoefficient float64, headPointAnomalyEl helpers.AnomalyElements,
	maxPacketSize float64, interfaceBufferSize int) IGroundStation {

	var newGS GroundStation

	newGS.Name = name
	newGS.Dt = dt
	newGS.TotalSimulationTime = totalSimulationTime
	newGS.TimeStamp = 0

	// Geo
	newGS.Latitude = latitude
	newGS.Longitude = longitude
	newGS.GSCalculation = groundStationCalculation
	newGS.HeadPointAnomaly = headPointAnomaly
	newGS.HeadPointAscension = headPointAscension
	newGS.HeadPointAnomalyEl = headPointAnomalyEl

	// Channels
	newGS.InterfaceBufferSize = interfaceBufferSize
	newGS.EventQueue = make(connections.PriorityQueue, 0)
	newGS.GSLInterfaceSample = connections.InitGSL(newGS.Name, speedOfLightVac, bandwidth, linkNoiseCoefficient, nil, 0.0,
		headPointAscension, headPointAnomalyEl, groundStationCalculation, maxPacketSize, interfaceBufferSize)
	newGS.GSLInterfaces = make(map[string]connections.INetworkInterface)
	newGS.PendingConnections = make([]LinkRequest, 0)

	return &newGS
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

func (gs *GroundStation) RunDistances() {
	log.Default().Println("Running ground station (Distance Mode): ", gs.Name)
	go startGSDistances(gs)
}

func (gs *GroundStation) SetDistanceLoggerChannel(channel *DistanceLoggerDeviceChannel) {
	gs.DistanceLoggerChannel = channel
}

func (gs *GroundStation) GetDistanceLoggerChannel() *DistanceLoggerDeviceChannel {
	return gs.DistanceLoggerChannel
}

func (gs *GroundStation) nextTimeStep() {
	gs.TimeStamp += gs.Dt
}

func (gs *GroundStation) updatePosition() {
	dt := float64(gs.Dt) * 0.001 // milliseconds to seconds
	gs.HeadPointAscension = gs.GSCalculation.UpdatePosition(gs.HeadPointAscension, dt)
}

func (gs *GroundStation) logDistances() {
	(*gs.DistanceLoggerChannel) <- UpdateDistancesMessage{
		DeviceName: gs.Name,
		TimeStamp:  gs.TimeStamp,
		Distances: gs.GSCalculation.FindSatellitesInRange(gs.Name, gs.HeadPointAscension, gs.HeadPointAnomalyEl,
			float64(gs.TimeStamp)*0.001),
	}
}

func startGSDistances(myGS IGroundStation) {
	for myGS.getTimeStamp() <= myGS.getTotalSimulationTime() {
		myGS.logDistances()
		myGS.nextTimeStep()
		myGS.updatePosition()
	}
	close(*myGS.GetDistanceLoggerChannel())
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

func (gs *GroundStation) SetForwardingTable(forwardingTable map[int]ForwardingEntry) {
	gs.ForwardingTable = forwardingTable
}

func (gs *GroundStation) GetNumberOfPackets() int {
	return gs.EventQueue.Len()
}

func (gs *GroundStation) SetLoggerChannel(channel *LoggerDeviceChannel) {
	gs.LoggerChannel = channel
}

func (gs *GroundStation) SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel) {
	gs.LinkerIncomingChannel = ingoingChannel
	gs.LinkerOutgoingChannel = outgoingChannel
}

func (gs *GroundStation) Run() {
	log.Default().Println("Running ground station: ", gs.Name)
	go startGS(gs)
}

func (gs *GroundStation) generatePackets(fromId int, maxPacketSize float64, entry TrafficEntry) ([]connections.Packet, int) {
	numberOfPackets := int(math.Ceil(1000.0 * entry.Length / maxPacketSize))
	packets := make([]connections.Packet, numberOfPackets)

	for i := 0; i < numberOfPackets; i++ {
		packets[i] = connections.Packet{
			PacketId:       fromId + i,
			Source:         gs.Name,
			Destination:    entry.Destination,
			Length:         maxPacketSize,
			PacketSentTime: float64(entry.TimeStamp),
		}
	}
	return packets, numberOfPackets
}

func (gs *GroundStation) GenerateTraffic(fromId int, traffic []TrafficEntry, maxPacketSize float64) (int, int) {
	totalNumberOfPackets := 0
	idAssigned := fromId
	for _, entry := range traffic {
		packets, numberOfPackets := gs.generatePackets(idAssigned, maxPacketSize, entry)
		totalNumberOfPackets += numberOfPackets
		idAssigned += numberOfPackets
		for index, packet := range packets {
			event := connections.Event{
				TimeStamp: float64(entry.TimeStamp),
				Type:      connections.SEND_EVENT,
				Data:      &packet,
			}
			item := connections.Item{
				Value: &event,
				Rank:  entry.TimeStamp,
				Index: index,
			}
			gs.EventQueue = append(gs.EventQueue, &item)
		}
	}
	heap.Init(&gs.EventQueue)

	return totalNumberOfPackets, idAssigned
}

func (gs *GroundStation) ProcessBuffers() {
	for _, inface := range gs.GSLInterfaces {
		if inface.HasSendChannel() {
			inface.ProcessBuffer()
		}
	}
}

func (gs *GroundStation) ReceiveFromInterfaces() {
	for gsName, inface := range gs.GSLInterfaces {
		if inface.GetDeviceConnectedTo() != "" {
			if inface.HasReceiveChannel() {
				receivedEvents := inface.Receive()
				for _, event := range receivedEvents {
					item := connections.Item{Value: &event, Rank: int(event.TimeStamp)}
					heap.Push(&gs.EventQueue, &item)
					gs.logEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, inface.GetDeviceConnectedTo(), gs.Name)
				}
			}
		} else {
			delete(gs.GSLInterfaces, gsName)
		}
	}
}

func (gs *GroundStation) logEvent(timeStamp int, eventType int, packet *connections.Packet, srcDevice string, destDevice string) {
	*gs.LoggerChannel <- SimulationEvent{
		TimeStamp:  timeStamp,
		EventType:  eventType,
		FromDevice: srcDevice,
		ToDevice:   destDevice,
		Packet:     packet,
	}
}

func (gs *GroundStation) findConnection(toSatellite string) connections.INetworkInterface {
	inface, ok := gs.GSLInterfaces[toSatellite]
	if ok {
		if inface.HasSendChannel() {
			return inface
		} else {
			gs.establishSendChannel(inface, toSatellite)
			return inface
		}
	} else {
		return gs.establishConnection(toSatellite)
	}
}

func (gs *GroundStation) establishSendChannel(inface connections.INetworkInterface, toSatellite string) {
	sendChannel := make(chan connections.Packet, gs.InterfaceBufferSize)
	inface.ChangeSendLink(toSatellite, &sendChannel)
	linkRequest := LinkRequest{
		ToDevice:    toSatellite,
		FromDevice:  gs.Name,
		SendChannel: &sendChannel,
	}
	select {
	case *gs.LinkerOutgoingChannel <- linkRequest:
		return
	default:
		gs.PendingConnections = append(gs.PendingConnections, linkRequest)
	}
}

func (gs *GroundStation) establishConnection(toSatellite string) connections.INetworkInterface {
	newNetworkInterface := gs.GSLInterfaceSample.Clone()
	gs.GSLInterfaces[toSatellite] = newNetworkInterface
	sendChannel := make(chan connections.Packet, gs.InterfaceBufferSize)
	newNetworkInterface.ChangeSendLink(toSatellite, &sendChannel)
	linkRequest := LinkRequest{
		ToDevice:    toSatellite,
		FromDevice:  gs.Name,
		SendChannel: &sendChannel,
	}
	select {
	case *gs.LinkerOutgoingChannel <- linkRequest:
		return newNetworkInterface
	default:
		gs.PendingConnections = append(gs.PendingConnections, linkRequest)
	}

	return newNetworkInterface
}

func (gs *GroundStation) SendPackets() {
	for !gs.EventQueue.IsEmpty() {
		itemPopped := heap.Pop(&gs.EventQueue).(*connections.Item)
		eventType := itemPopped.Value.Type
		if eventType == connections.SEND_EVENT {
			packet := *itemPopped.Value.Data
			if packet.Destination != gs.Name {
				timeStamp := int(itemPopped.Value.TimeStamp/float64(gs.Dt)) * gs.Dt
				forwardingSatellite := gs.ForwardingTable[timeStamp][packet.Destination]
				if forwardingSatellite == "" {
					println("No forwarding choice found for packet: ", packet.PacketId, " at time: ", timeStamp, " with destination: ", packet.Destination, " and source: ", gs.Name)
				}
				connection := gs.findConnection(forwardingSatellite)
				packetDropped, timeOfAttempt := connection.Send(packet, itemPopped.Value.TimeStamp)
				if !packetDropped {
					gs.logEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, gs.Name, connection.GetDeviceConnectedTo())
				} else {
					gs.logEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, gs.Name, connection.GetDeviceConnectedTo())
				}
			} else {
				gs.logEvent(int(itemPopped.Value.TimeStamp), SIMULATION_EVENT_DELIVERED, itemPopped.Value.Data, packet.Source, packet.Destination)
			}
		}
	}
}

func (gs *GroundStation) CheckIncomingConnection() {
	select {
	case linkReq := <-*gs.LinkerIncomingChannel:
		inface, found := gs.GSLInterfaces[linkReq.FromDevice]
		if found {
			inface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
		} else {
			newInterface := gs.GSLInterfaceSample.Clone()
			newInterface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
			gs.GSLInterfaces[linkReq.FromDevice] = newInterface
		}
	default:
		return
	}
}

func (gs *GroundStation) SendPendingRequests() {
	indx := 0
	for indx < len(gs.PendingConnections) {
		select {
		case *gs.LinkerOutgoingChannel <- gs.PendingConnections[indx]:
			gs.PendingConnections = append(gs.PendingConnections[:indx], gs.PendingConnections[indx+1:]...)
		default:
			indx++
		}
	}
}

func startGS(myGS IGroundStation) {
	for {
		myGS.CheckIncomingConnection()
		myGS.ReceiveFromInterfaces()
		myGS.SendPendingRequests()
		myGS.SendPackets()
		myGS.ProcessBuffers()
	}
}
