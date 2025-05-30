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
	GSLInterface         connections.INetworkInterface
	DistanceSpaceChannel *DistanceSpaceDeviceChannel
	SpaceChannel         *SpaceDeviceChannel
}

type IGroundStation interface {
	getTimeStamp() int
	getTotalSimulationTime() int
	GetName() string
	// Distance Mode
	RunDistances()
	nextTimeStep()
	updatePosition()
	updateSpaceOnDistances()
	SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel)
	GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel
	// Simulation Mode
	Run()
	GetSpaceChannel() *SpaceDeviceChannel
	SetSpaceChannel(channel *SpaceDeviceChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	GenerateTraffic(fromId int, traffic []TrafficEntry, maxPacketSize float64) int
	ReceiveFromInterfaces()
	SendPackets()
	CheckIncomingConnections() bool
	generatePackets(fromId int, maxPacketSize float64, entry TrafficEntry) []connections.Packet
	sendEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string)
	establishConnection(toSatellite string, timeStamp int)
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
	newGS.EventQueue = make(connections.PriorityQueue, 0)
	newGS.GSLInterface = connections.InitGSL(newGS.Name, speedOfLightVac, bandwidth, linkNoiseCoefficient, nil, 0.0,
		headPointAscension, headPointAnomalyEl, groundStationCalculation, maxPacketSize*float64(interfaceBufferSize))

	return &newGS
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

func (gs *GroundStation) RunDistances() {
	log.Default().Println("Running ground station (Distance Mode): ", gs.Name)
	go startGSDistances(gs)
}

func (gs *GroundStation) SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel) {
	gs.DistanceSpaceChannel = channel
}

func (gs *GroundStation) GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel {
	return gs.DistanceSpaceChannel
}

func (gs *GroundStation) nextTimeStep() {
	gs.TimeStamp += gs.Dt
}

func (gs *GroundStation) updatePosition() {
	dt := float64(gs.Dt) * 0.001 // milliseconds to seconds
	gs.HeadPointAscension = gs.GSCalculation.UpdatePosition(gs.HeadPointAscension, dt)
}

func (gs *GroundStation) updateSpaceOnDistances() {
	(*gs.DistanceSpaceChannel) <- UpdateDistancesMessage{
		DeviceName: gs.Name,
		TimeStamp:  gs.TimeStamp,
		Distances: gs.GSCalculation.FindSatellitesInRange(gs.Name, gs.HeadPointAscension, gs.HeadPointAnomalyEl,
			float64(gs.TimeStamp)*0.001),
	}
}

func startGSDistances(myGS IGroundStation) {
	for myGS.getTimeStamp() <= myGS.getTotalSimulationTime() {
		myGS.updateSpaceOnDistances()
		myGS.nextTimeStep()
		myGS.updatePosition()
	}
	close(*myGS.GetDistanceSpaceChannel())
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

func (gs *GroundStation) SetForwardingTable(forwardingTable map[int]ForwardingEntry) {
	gs.ForwardingTable = forwardingTable
}

func (gs *GroundStation) GetSpaceChannel() *SpaceDeviceChannel {
	return gs.SpaceChannel
}

func (gs *GroundStation) SetSpaceChannel(channel *SpaceDeviceChannel) {
	gs.SpaceChannel = channel
}

func (gs *GroundStation) Run() {
	log.Default().Println("Running ground station: ", gs.Name)
	go startGS(gs)
}

func (gs *GroundStation) generatePackets(fromId int, maxPacketSize float64, entry TrafficEntry) []connections.Packet {
	numberOfPackets := int(math.Ceil(1000 * entry.Length / maxPacketSize))
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
	return packets
}

func (gs *GroundStation) GenerateTraffic(fromId int, traffic []TrafficEntry, maxPacketSize float64) int {
	number_of_packets := 0
	id_assigned := fromId
	for _, entry := range traffic {
		packets := gs.generatePackets(id_assigned, maxPacketSize, entry)
		number_of_packets += len(packets)
		id_assigned += len(packets)
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

	return number_of_packets
}

func (gs *GroundStation) ReceiveFromInterfaces() {
	if gs.GSLInterface.GetDeviceConnectedTo() != "" {
		receivedEvents := gs.GSLInterface.Receive()
		for _, event := range receivedEvents {
			item := connections.Item{Value: &event, Rank: int(event.TimeStamp)}
			heap.Push(&gs.EventQueue, &item)
			gs.sendEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, gs.GSLInterface.GetDeviceConnectedTo(), gs.Name)
		}
	}
}

func (gs *GroundStation) sendEvent(timeStamp int, eventType int, packet *connections.Packet, srcDevice string, destDevice string) {
	*gs.SpaceChannel <- SimulationEvent{
		TimeStamp:  timeStamp,
		EventType:  eventType,
		FromDevice: srcDevice,
		ToDevice:   destDevice,
		Packet:     packet,
	}
}

func (gs *GroundStation) establishConnection(toSatellite string, timeStamp int) {
	*gs.SpaceChannel <- SimulationEvent{
		TimeStamp:  timeStamp,
		EventType:  SIMULATION_EVENT_CONNECTION_ESTABLISHED,
		FromDevice: gs.Name,
		ToDevice:   toSatellite,
		Packet:     nil,
		LinkReq:    nil,
	}
	connectionEvent := <-*gs.SpaceChannel
	gs.GSLInterface.ChangeLink(connectionEvent.ToDevice, connectionEvent.LinkReq.SendChannel, connectionEvent.LinkReq.RecieveChannel)
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
				if gs.GSLInterface.GetDeviceConnectedTo() != forwardingSatellite {
					gs.establishConnection(forwardingSatellite, timeStamp)
				}
				packetDropped, packetBuffered, timeOfAttempt := gs.GSLInterface.Send(packet, itemPopped.Value.TimeStamp)
				if !packetDropped && !packetBuffered {
					gs.sendEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, gs.Name, gs.GSLInterface.GetDeviceConnectedTo())
				} else if packetDropped {
					gs.sendEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, gs.Name, gs.GSLInterface.GetDeviceConnectedTo())
				} else if packetBuffered {
					heap.Push(&gs.EventQueue, itemPopped)
					break
				}
			} else {
				gs.sendEvent(int(itemPopped.Value.TimeStamp), SIMULATION_EVENT_DELIVERED, itemPopped.Value.Data, packet.Source, packet.Destination)
			}
		}
	}
}

func (gs *GroundStation) CheckIncomingConnections() bool {
	select {
	case event, ok := <-*gs.SpaceChannel:
		if !ok {
			return true
		}
		if event.EventType == SIMULATION_EVENT_CONNECTION_ESTABLISHED {
			gs.GSLInterface.ChangeLink(event.ToDevice, event.LinkReq.SendChannel, event.LinkReq.RecieveChannel)
		}
		return false
	default:
		return false
	}
}

func startGS(myGS IGroundStation) {
	simulationDone := false
	for !simulationDone {
		simulationDone = myGS.CheckIncomingConnections()
		myGS.ReceiveFromInterfaces()
		myGS.SendPackets()
	}
}
