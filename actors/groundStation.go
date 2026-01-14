package actors

import (
	"container/heap"
	"log"
	"math"
	"reflect"

	"Starfield/connections"
	"Starfield/helpers"
)

type TrafficEntry struct {
	Destination string
	TimeStamp   int     // in milliseconds
	Length      float64 // in Mb
}

type GroundStation struct {
	// General
	Name                string
	Dt                  float64 // in milliseconds
	DistancesTimeStamp  float64 // in milliseconds
	TimeStamp           float64 // in milliseconds
	TotalSimulationTime float64 // in milliseconds

	// Geometrical parameters
	Longitude          float64
	Latitude           float64
	HeadPointAnomaly   float64
	HeadPointAscension float64
	HeadPointAnomalyEl helpers.AnomalyElements
	GSCalculation      helpers.IGroundStationCalculation

	// Packet Level Simulation
	ForwardingTable  map[int]ForwardingEntry
	EventQueue       connections.PriorityQueue
	lastAckTimeStamp float64 // in milliseconds

	// Goroutines and connections, and channels
	InterfaceBufferSize            int
	GSLInterfaceSample             connections.INetworkInterface
	GSLInterfaces                  map[string]connections.INetworkInterface
	DistanceLoggerChannel          *DistanceLoggerDeviceChannel
	SphericalPositionLoggerChannel *SphericalPositionLoggerDeviceChannel
	CartesianPositionLoggerChannel *CartesianPositionLoggerDeviceChannel
	LoggerChannel                  *LoggerDeviceChannel
	ProgressTokenChannel           *ProgressTokenChannel
	AckTokenChannel                *AckTokenChannel
	LinkerOutgoingChannel          *LinkRequestChannel
	LinkerIncomingChannel          *LinkRequestChannel
	PendingConnections             []LinkRequest
}

type IGroundStation interface {
	getDistancesTimeStamp() float64
	getTotalSimulationTime() float64
	GetName() string
	// Position Mode
	RunCartesianPositions()
	RunSphericalPositions()
	GetSphericalPositionLoggerChannel() *SphericalPositionLoggerDeviceChannel
	GetCartesianPositionLoggerChannel() *CartesianPositionLoggerDeviceChannel
	SetSphericalPositionLoggerChannel(channel *SphericalPositionLoggerDeviceChannel)
	SetCartesianPositionLoggerChannel(channel *CartesianPositionLoggerDeviceChannel)
	logSphericalPosition()
	logCartesianPosition()
	// Distance Mode
	RunDistances()
	nextTimeStep()
	updatePosition()
	logDistances()
	SetDistanceLoggerChannel(channel *DistanceLoggerDeviceChannel)
	GetDistanceLoggerChannel() *DistanceLoggerDeviceChannel
	// Simulation Mode
	Run()
	SetLoggerChannel(channel *LoggerDeviceChannel)
	SetProgressTokenChannel(channel *ProgressTokenChannel)
	SetAckTokenChannel(channel *AckTokenChannel)
	SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	GenerateTraffic(fromId int, traffic []TrafficEntry, maxPacketSize float64) (int, int)
	ReceiveFromInterfaces()
	SendPackets() float64
	ProcessBuffers()
	SendTimeStampAck(nextTimeStamp float64)
	SendPendingRequests()
	ProcessIncomingConnection(linkReq LinkRequest)
	CheckIncomingConnections()
	InitChannelCases(selectCases *[]reflect.SelectCase) []string
	WatchEvents()
	IsBlocking() bool
	areAllBuffersEmpty() bool
	getReceiveGSL() ([]*chan connections.Packet, []string)
	generatePackets(fromId int, maxPacketSize float64, entry TrafficEntry) ([]connections.Packet, int)
	logEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string)
	findConnection(toSatellite string) connections.INetworkInterface
	establishConnection(toSatellite string) connections.INetworkInterface
	establishSendChannel(inface connections.INetworkInterface, toSatellite string)
}

func (gs *GroundStation) GetName() string {
	return gs.Name
}

func (gs *GroundStation) getDistancesTimeStamp() float64 {
	return gs.DistancesTimeStamp
}

func (gs *GroundStation) getTotalSimulationTime() float64 {
	return gs.TotalSimulationTime
}

func NewGroundStation(name string, latitude float64, longitude float64, dt float64, totalSimulationTime float64,
	headPointAnomaly float64, headPointAscension float64, groundStationCalculation helpers.IGroundStationCalculation,
	speedOfLightVac float64, bandwidth float64, linkNoiseCoefficient float64, headPointAnomalyEl helpers.AnomalyElements,
	maxPacketSize float64, interfaceBufferSize int) IGroundStation {

	var newGS GroundStation

	newGS.Name = name
	newGS.Dt = dt
	newGS.TotalSimulationTime = totalSimulationTime
	newGS.DistancesTimeStamp = 0.0
	newGS.TimeStamp = -1.0

	// Geo
	newGS.Latitude = latitude
	newGS.Longitude = longitude
	newGS.GSCalculation = groundStationCalculation
	newGS.HeadPointAnomaly = headPointAnomaly
	newGS.HeadPointAscension = headPointAscension
	newGS.HeadPointAnomalyEl = headPointAnomalyEl

	// Packet Level Simulation
	newGS.lastAckTimeStamp = -1.0

	// Channels
	newGS.InterfaceBufferSize = interfaceBufferSize
	newGS.EventQueue = make(connections.PriorityQueue, 0)
	newGS.GSLInterfaceSample = connections.InitGSL(newGS.Name, speedOfLightVac, bandwidth, linkNoiseCoefficient, nil, 0.0,
		headPointAscension, headPointAnomalyEl, groundStationCalculation, maxPacketSize, interfaceBufferSize)
	newGS.GSLInterfaces = make(map[string]connections.INetworkInterface)
	newGS.PendingConnections = make([]LinkRequest, 0)

	return &newGS
}

//////////////////////////////////// ****** Positions Mode ****** //////////////////////////////////////////////////

func (gs *GroundStation) RunSphericalPositions() {
	log.Default().Println("Running ground station (Spherical position Mode): ", gs.Name)
	go startSphericalGroundStationPositions(gs)
}

func (gs *GroundStation) RunCartesianPositions() {
	log.Default().Println("Running ground station (Cartesian position Mode): ", gs.Name)
	go startCartesianGroundStationPositions(gs)
}

func (gs *GroundStation) GetSphericalPositionLoggerChannel() *SphericalPositionLoggerDeviceChannel {
	return gs.SphericalPositionLoggerChannel
}

func (gs *GroundStation) GetCartesianPositionLoggerChannel() *CartesianPositionLoggerDeviceChannel {
	return gs.CartesianPositionLoggerChannel
}

func (gs *GroundStation) SetSphericalPositionLoggerChannel(channel *SphericalPositionLoggerDeviceChannel) {
	gs.SphericalPositionLoggerChannel = channel
}

func (gs *GroundStation) SetCartesianPositionLoggerChannel(channel *CartesianPositionLoggerDeviceChannel) {
	gs.CartesianPositionLoggerChannel = channel
}

func (gs *GroundStation) logSphericalPosition() {
	adjustedLongitude := math.Mod(gs.Longitude+2.0*math.Pi, 2.0*math.Pi)
	if adjustedLongitude > math.Pi {
		adjustedLongitude -= 2.0 * math.Pi
	}
	sphericalCoordinates := helpers.SphericalCoordinates{
		Radius:    gs.GSCalculation.GetEarthRadius(),
		Latitude:  gs.Latitude,
		Longitude: adjustedLongitude,
	}

	(*gs.SphericalPositionLoggerChannel) <- UpdateSphericalPositionMessage{
		DeviceName: gs.Name,
		TimeStamp:  gs.DistancesTimeStamp,
		Spherical:  sphericalCoordinates,
	}
}

func (gs *GroundStation) logCartesianPosition() {
	cartesianCoordinates := helpers.ConvertToCartesianFromSpherical(
		helpers.SphericalCoordinates{
			Radius:    gs.GSCalculation.GetEarthRadius(),
			Latitude:  gs.Latitude,
			Longitude: gs.Longitude,
		},
	)

	(*gs.CartesianPositionLoggerChannel) <- UpdateCartesianPositionMessage{
		DeviceName: gs.Name,
		TimeStamp:  gs.DistancesTimeStamp,
		Cartesian:  cartesianCoordinates,
	}
}

func startSphericalGroundStationPositions(myGS IGroundStation) {
	for myGS.getDistancesTimeStamp() <= myGS.getTotalSimulationTime() {
		myGS.logSphericalPosition()
		myGS.nextTimeStep()
		myGS.updatePosition()
	}
	close(*myGS.GetSphericalPositionLoggerChannel())
}

func startCartesianGroundStationPositions(myGS IGroundStation) {
	for myGS.getDistancesTimeStamp() <= myGS.getTotalSimulationTime() {
		myGS.logCartesianPosition()
		myGS.nextTimeStep()
		myGS.updatePosition()
	}
	close(*myGS.GetCartesianPositionLoggerChannel())
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
	gs.DistancesTimeStamp += gs.Dt
}

func (gs *GroundStation) updatePosition() {
	dt := float64(gs.Dt) * 0.001 // milliseconds to seconds
	gs.Longitude = gs.GSCalculation.UpdatePosition(gs.Longitude, dt)
	gs.HeadPointAscension = gs.GSCalculation.UpdatePosition(gs.HeadPointAscension, dt)
}

func (gs *GroundStation) logDistances() {
	(*gs.DistanceLoggerChannel) <- UpdateDistancesMessage{
		DeviceName: gs.Name,
		TimeStamp:  gs.DistancesTimeStamp,
		Distances: gs.GSCalculation.FindSatellitesInRange(gs.Name, gs.HeadPointAscension, gs.HeadPointAnomalyEl,
			gs.DistancesTimeStamp*0.001),
	}
}

func startGSDistances(myGS IGroundStation) {
	for myGS.getDistancesTimeStamp() <= myGS.getTotalSimulationTime() {
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

func (gs *GroundStation) SetLoggerChannel(channel *LoggerDeviceChannel) {
	gs.LoggerChannel = channel
}

func (gs *GroundStation) SetProgressTokenChannel(channel *ProgressTokenChannel) {
	gs.ProgressTokenChannel = channel
}

func (gs *GroundStation) SetAckTokenChannel(channel *AckTokenChannel) {
	gs.AckTokenChannel = channel
}

func (gs *GroundStation) SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel) {
	gs.LinkerIncomingChannel = ingoingChannel
	gs.LinkerOutgoingChannel = outgoingChannel
}

func (gs *GroundStation) areAllBuffersEmpty() bool {
	for _, inface := range gs.GSLInterfaces {
		if inface.HasSendChannel() && inface.IsBufferNotEmpty() {
			return false
		}
	}
	return true
}

func (gs *GroundStation) IsBlocking() bool {
	return len(gs.PendingConnections) == 0 &&
		gs.areAllBuffersEmpty() &&
		(len(gs.EventQueue) == 0 || gs.TimeStamp <= gs.lastAckTimeStamp)
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

func (gs *GroundStation) SendPackets() float64 {
	nextEventTime := gs.TotalSimulationTime
	for !gs.EventQueue.IsEmpty() {
		itemPopped := heap.Pop(&gs.EventQueue).(*connections.Item)
		eventType := itemPopped.Value.Type
		nextEventTime = itemPopped.Value.TimeStamp
		if nextEventTime > gs.TimeStamp {
			heap.Push(&gs.EventQueue, itemPopped)
			break
		}
		if eventType == connections.SEND_EVENT {
			packet := *itemPopped.Value.Data
			if packet.Destination != gs.Name {
				roundedTimeStamp := int(nextEventTime/gs.Dt) * int(gs.Dt)
				forwardingSatellite := gs.ForwardingTable[roundedTimeStamp][packet.Destination]
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
		nextEventTime = gs.TotalSimulationTime
	}
	return nextEventTime
}

func (gs *GroundStation) CheckIncomingConnections() {
	channelEmpty := false
	for !channelEmpty {
		select {
		case linkReq := <-*gs.LinkerIncomingChannel:
			gs.ProcessIncomingConnection(linkReq)
		default:
			channelEmpty = true
		}
	}
}

func (gs *GroundStation) ProcessIncomingConnection(linkReq LinkRequest) {
	inface, found := gs.GSLInterfaces[linkReq.FromDevice]
	if found {
		inface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
	} else {
		newInterface := gs.GSLInterfaceSample.Clone()
		newInterface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
		gs.GSLInterfaces[linkReq.FromDevice] = newInterface
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

func (gs *GroundStation) SendTimeStampAck(nextTimeStamp float64) {
	if gs.lastAckTimeStamp < gs.TimeStamp && gs.TimeStamp < nextTimeStamp {
		*gs.AckTokenChannel <- AckToken{
			TimeStampAck:  gs.TimeStamp,
			NextTimeStamp: nextTimeStamp,
		}
		gs.lastAckTimeStamp = gs.TimeStamp
	}
}

func (gs *GroundStation) getReceiveGSL() ([]*chan connections.Packet, []string) {
	channels := make([]*chan connections.Packet, 0)
	channelOwners := make([]string, 0)
	for gsName, inface := range gs.GSLInterfaces {
		if inface.GetDeviceConnectedTo() != "" {
			if inface.HasReceiveChannel() {
				channels = append(channels, inface.GetReceiveChannel())
				channelOwners = append(channelOwners, gsName)
			}
		} else {
			delete(gs.GSLInterfaces, gsName)
		}
	}
	return channels, channelOwners
}

func (gs *GroundStation) InitChannelCases(selectCases *[]reflect.SelectCase) []string {
	channels, channelOwners := gs.getReceiveGSL()
	*selectCases = make([]reflect.SelectCase, len(channels)+2)
	(*selectCases)[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*gs.ProgressTokenChannel)}
	(*selectCases)[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*gs.LinkerIncomingChannel)}
	for i, channel := range channels {
		(*selectCases)[i+2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
	return channelOwners
}

func (gs *GroundStation) WatchEvents() {
	var selectIncomingChannelsCases []reflect.SelectCase
	channelOwners := gs.InitChannelCases(&selectIncomingChannelsCases)
	indx, value, _ := reflect.Select(selectIncomingChannelsCases)
	switch indx {
	case 0:
		token := value.Interface().(ProgressToken)
		gs.TimeStamp = max(gs.TimeStamp, token.TimeStamp)
	case 1:
		linkReq := value.Interface().(LinkRequest)
		gs.ProcessIncomingConnection(linkReq)
	default:
		packet := value.Interface().(connections.Packet)
		inface := gs.GSLInterfaces[channelOwners[indx-2]]
		event := inface.ProcessReceivedPacket(&packet)
		item := connections.Item{Value: &event, Rank: int(event.TimeStamp)}
		heap.Push(&gs.EventQueue, &item)
		gs.logEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, inface.GetDeviceConnectedTo(), gs.Name)
	}
}

func startGS(myGS IGroundStation) {
	for {
		if myGS.IsBlocking() {
			myGS.WatchEvents()
		}
		myGS.CheckIncomingConnections()
		myGS.ReceiveFromInterfaces()
		myGS.SendPendingRequests()
		nextTimeStamp := myGS.SendPackets()
		myGS.ProcessBuffers()
		myGS.SendTimeStampAck(nextTimeStamp)
	}
}
