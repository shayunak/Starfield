package actors

import (
	"container/heap"
	"fmt"
	"log"
	"math"
	"reflect"

	"Starfield/connections"

	"Starfield/helpers"
)

type ForwardingEntry map[string]string

type Satellite struct {
	// General
	Name                string
	Id                  int
	Dt                  float64 // in milliseconds
	DistancesTimeStamp  float64 // in milliseconds
	TimeStamp           float64 // in milliseconds
	TotalSimulationTime float64 // in milliseconds

	// Geometrical parameters
	AnomalyElements           helpers.AnomalyElements
	Orbit                     helpers.IOrbit
	OrbitalAnomaly            float64 // in radians
	AnomalyCalculations       helpers.IAnomalyCalculation
	GroundStationCalculations helpers.IGroundStationCalculation

	// Packet Level Simulation
	ForwardingTable  map[int]ForwardingEntry
	EventQueue       connections.PriorityQueue
	lastAckTimeStamp float64 // in milliseconds

	// Goroutines and connections, and channels
	InterfaceBufferSize            int
	ISLInterfaceSample             connections.INetworkInterface
	ISLInterfaces                  map[string]connections.INetworkInterface
	GSLInterfaceSample             connections.INetworkInterface
	GSLInterfaces                  map[string]connections.INetworkInterface
	LinkerOutgoingChannel          *LinkRequestChannel
	LinkerIncomingChannel          *LinkRequestChannel
	DistanceLoggerChannel          *DistanceLoggerDeviceChannel
	SphericalPositionLoggerChannel *SphericalPositionLoggerDeviceChannel
	CartesianPositionLoggerChannel *CartesianPositionLoggerDeviceChannel
	LoggerChannel                  *LoggerDeviceChannel
	ProgressTokenChannel           *ProgressTokenChannel
	AckTokenChannel                *AckTokenChannel
	PendingConnections             []LinkRequest
}

type ISatellite interface {
	// General
	GetName() string
	getDistancesTimeStamp() float64
	getTotalSimulationTime() float64
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
	GetDistanceLoggerChannel() *DistanceLoggerDeviceChannel
	SetDistanceLoggerChannel(channel *DistanceLoggerDeviceChannel)
	updatePosition()
	logDistances()
	nextTimeStep()
	findSatellitesInRange() map[string]float64
	findGroundStationsInRange() map[string]float64
	// Simulation Mode
	GetNumberOfPackets() int
	Run()
	AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	SetLoggerChannel(channel *LoggerDeviceChannel)
	SetProgressTokenChannel(channel *ProgressTokenChannel)
	SetAckTokenChannel(channel *AckTokenChannel)
	SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	ReceiveFromInterfaces()
	SendPackets() float64
	SendTimeStampAck(nextTimeStamp float64)
	SendPendingRequests()
	ProcessBuffers()
	ProcessIncomingConnection(linkReq LinkRequest)
	CheckIncomingConnections()
	InitChannelCases(selectCases *[]reflect.SelectCase) ([]string, []string)
	WatchEvents()
	IsBlocking() bool
	areAllBuffersEmpty() bool
	getReceiveGSL() ([]*chan connections.Packet, []string)
	getReceiveISL() ([]*chan connections.Packet, []string)
	findGSLConnection(toGroundStation string) connections.INetworkInterface
	findISLConnection(toSatellite string) connections.INetworkInterface
	getISLInterfaceNames() []string
	establishGSLConnection(toGroundStation string) connections.INetworkInterface
	establishISLConnection(toSatellite string) connections.INetworkInterface
	establishSendChannel(inface connections.INetworkInterface, toDevice string)
	logEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string)
}

func (satellite *Satellite) GetName() string {
	return satellite.Name
}

func (satellite *Satellite) getDistancesTimeStamp() float64 {
	return satellite.DistancesTimeStamp
}

func (satellite *Satellite) getTotalSimulationTime() float64 {
	return satellite.TotalSimulationTime
}

func NewSatellite(id int, orbitalPhase float64, dt float64, totalSimulationTime float64, orbit helpers.IOrbit,
	anomalyCalculations helpers.IAnomalyCalculation, groundStationCalculations helpers.IGroundStationCalculation,
	numberOfIsls int, speedOfLightVac float64, ISLBandwidth float64, ISLLinkNoiseCoefficient float64,
	GSLBandwidth float64, GSLLinkNoiseCoefficient float64, acquisitionTime float64, maxPacketSize float64,
	interfaceBufferSize int) ISatellite {
	var newSatellite Satellite

	newSatellite.Id = id
	newSatellite.Name = fmt.Sprintf("%s-%d", orbit.GetOrbitId(), id)
	newSatellite.Dt = dt
	newSatellite.TotalSimulationTime = totalSimulationTime
	newSatellite.DistancesTimeStamp = 0.0
	newSatellite.TimeStamp = -1.0

	// Geo
	newSatellite.OrbitalAnomaly = orbitalPhase * (math.Pi / 180.0)
	newSatellite.AnomalyCalculations = anomalyCalculations
	newSatellite.GroundStationCalculations = groundStationCalculations
	newSatellite.Orbit = orbit
	newSatellite.AnomalyElements = helpers.AnomalyElements{
		AnomalySinus:   math.Sin(newSatellite.OrbitalAnomaly),
		AnomalyCosinus: math.Cos(newSatellite.OrbitalAnomaly),
	}

	// Packet Level Simulation
	newSatellite.lastAckTimeStamp = -1.0

	// Channels
	newSatellite.InterfaceBufferSize = interfaceBufferSize
	newSatellite.EventQueue = make(connections.PriorityQueue, 0)
	heap.Init(&newSatellite.EventQueue)
	newSatellite.ISLInterfaces = make(map[string]connections.INetworkInterface)
	newSatellite.ISLInterfaceSample = connections.InitISL(newSatellite.Name, 0, speedOfLightVac, ISLBandwidth,
		ISLLinkNoiseCoefficient, anomalyCalculations, maxPacketSize, interfaceBufferSize)
	newSatellite.GSLInterfaceSample = connections.InitGSL(newSatellite.Name, speedOfLightVac, GSLBandwidth, GSLLinkNoiseCoefficient, orbit,
		newSatellite.OrbitalAnomaly, 0.0, newSatellite.AnomalyElements, groundStationCalculations, maxPacketSize, interfaceBufferSize)
	newSatellite.GSLInterfaces = make(map[string]connections.INetworkInterface)
	newSatellite.PendingConnections = make([]LinkRequest, 0)

	return &newSatellite
}

//////////////////////////////////// ****** Positions Mode ****** //////////////////////////////////////////////////

func (satellite *Satellite) RunSphericalPositions() {
	log.Default().Println("Running satellite (Spherical position Mode): ", satellite.Name)
	go startSphericalSatellitePositions(satellite)
}

func (satellite *Satellite) RunCartesianPositions() {
	log.Default().Println("Running satellite (Cartesian position Mode): ", satellite.Name)
	go startCartesianSatellitePositions(satellite)
}

func (satellite *Satellite) GetSphericalPositionLoggerChannel() *SphericalPositionLoggerDeviceChannel {
	return satellite.SphericalPositionLoggerChannel
}

func (satellite *Satellite) GetCartesianPositionLoggerChannel() *CartesianPositionLoggerDeviceChannel {
	return satellite.CartesianPositionLoggerChannel
}

func (satellite *Satellite) SetSphericalPositionLoggerChannel(channel *SphericalPositionLoggerDeviceChannel) {
	satellite.SphericalPositionLoggerChannel = channel
}

func (satellite *Satellite) SetCartesianPositionLoggerChannel(channel *CartesianPositionLoggerDeviceChannel) {
	satellite.CartesianPositionLoggerChannel = channel
}

func (satellite *Satellite) logSphericalPosition() {
	sphericalCoordinates := helpers.ConvertToSpherical(
		helpers.ConvertToCartesian(
			helpers.KepplerianCoordinates{
				Anomaly:     satellite.OrbitalAnomaly,
				Radius:      satellite.Orbit.GetRadius(),
				Ascension:   satellite.Orbit.GetAscension(),
				Inclination: satellite.Orbit.GetInclination(),
			},
		),
	)

	(*satellite.SphericalPositionLoggerChannel) <- UpdateSphericalPositionMessage{
		DeviceName: satellite.Name,
		TimeStamp:  satellite.DistancesTimeStamp,
		Spherical:  sphericalCoordinates,
	}
}

func (satellite *Satellite) logCartesianPosition() {
	cartesianCoordinates := helpers.ConvertToCartesian(
		helpers.KepplerianCoordinates{
			Anomaly:     satellite.OrbitalAnomaly,
			Radius:      satellite.Orbit.GetRadius(),
			Ascension:   satellite.Orbit.GetAscension(),
			Inclination: satellite.Orbit.GetInclination(),
		},
	)

	(*satellite.CartesianPositionLoggerChannel) <- UpdateCartesianPositionMessage{
		DeviceName: satellite.Name,
		TimeStamp:  satellite.DistancesTimeStamp,
		Cartesian:  cartesianCoordinates,
	}
}

func startSphericalSatellitePositions(mySatellite ISatellite) {
	for mySatellite.getDistancesTimeStamp() <= mySatellite.getTotalSimulationTime() {
		mySatellite.logSphericalPosition()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
	close(*mySatellite.GetSphericalPositionLoggerChannel())
}

func startCartesianSatellitePositions(mySatellite ISatellite) {
	for mySatellite.getDistancesTimeStamp() <= mySatellite.getTotalSimulationTime() {
		mySatellite.logCartesianPosition()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
	close(*mySatellite.GetCartesianPositionLoggerChannel())
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

func (satellite *Satellite) RunDistances() {
	log.Default().Println("Running satellite (Distance Mode): ", satellite.Name)
	go startSatelliteDistances(satellite)
}

func (satellite *Satellite) GetDistanceLoggerChannel() *DistanceLoggerDeviceChannel {
	return satellite.DistanceLoggerChannel
}

func (satellite *Satellite) SetDistanceLoggerChannel(channel *DistanceLoggerDeviceChannel) {
	satellite.DistanceLoggerChannel = channel
}

func (satellite *Satellite) updatePosition() {
	dt := float64(satellite.Dt) * 0.001 // milliseconds to seconds
	satellite.OrbitalAnomaly, satellite.AnomalyElements = satellite.AnomalyCalculations.UpdatePosition(satellite.OrbitalAnomaly, dt)
}

func (satellite *Satellite) nextTimeStep() {
	satellite.DistancesTimeStamp += satellite.Dt
}

func mergeMaps(satelliteMap map[string]float64, groundStationMap map[string]float64) map[string]float64 {
	mergedMap := make(map[string]float64)

	for key, value := range satelliteMap {
		mergedMap[key] = value
	}
	for key, value := range groundStationMap {
		mergedMap[key] = value
	}

	return mergedMap
}

func (satellite *Satellite) findSatellitesInRange() map[string]float64 {
	satelliteOrbitalAscension := satellite.Orbit.GetAscension()
	lengthLimitRatio := satellite.AnomalyCalculations.GetLengthLimitRatio()
	return satellite.AnomalyCalculations.FindSatellitesInRange(satellite.Name, lengthLimitRatio, satellite.AnomalyElements,
		satelliteOrbitalAscension, satellite.DistancesTimeStamp*0.001)
}

func (satellite *Satellite) findGroundStationsInRange() map[string]float64 {
	return satellite.GroundStationCalculations.GetCoveringGroundStations(satellite.DistancesTimeStamp*0.001, satellite.OrbitalAnomaly,
		satellite.Orbit)
}

func (satellite *Satellite) logDistances() {
	satelliteDistances := satellite.findSatellitesInRange()
	groundStationDistances := satellite.findGroundStationsInRange()

	(*satellite.DistanceLoggerChannel) <- UpdateDistancesMessage{
		DeviceName: satellite.Name,
		TimeStamp:  satellite.DistancesTimeStamp,
		Distances:  mergeMaps(satelliteDistances, groundStationDistances),
	}
}

func startSatelliteDistances(mySatellite ISatellite) {
	for mySatellite.getDistancesTimeStamp() <= mySatellite.getTotalSimulationTime() {
		mySatellite.logDistances()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
	close(*mySatellite.GetDistanceLoggerChannel())
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

func (satellite *Satellite) GetNumberOfPackets() int {
	return satellite.EventQueue.Len()
}

func (satellite *Satellite) SetForwardingTable(forwardingTable map[int]ForwardingEntry) {
	satellite.ForwardingTable = forwardingTable
}

func (satellite *Satellite) SetLoggerChannel(channel *LoggerDeviceChannel) {
	satellite.LoggerChannel = channel
}

func (satellite *Satellite) SetProgressTokenChannel(channel *ProgressTokenChannel) {
	satellite.ProgressTokenChannel = channel
}

func (satellite *Satellite) SetAckTokenChannel(channel *AckTokenChannel) {
	satellite.AckTokenChannel = channel
}

func (satellite *Satellite) SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel) {
	satellite.LinkerIncomingChannel = ingoingChannel
	satellite.LinkerOutgoingChannel = outgoingChannel
}

func (satellite *Satellite) areAllBuffersEmpty() bool {
	for _, inface := range satellite.GSLInterfaces {
		if inface.HasSendChannel() && inface.IsBufferNotEmpty() {
			return false
		}
	}
	for _, inface := range satellite.ISLInterfaces {
		if inface.HasSendChannel() && inface.IsBufferNotEmpty() {
			return false
		}
	}
	return true
}

func (satellite *Satellite) IsBlocking() bool {
	return len(satellite.PendingConnections) == 0 &&
		satellite.areAllBuffersEmpty() &&
		(len(satellite.EventQueue) == 0 || satellite.TimeStamp <= satellite.lastAckTimeStamp)
}

func (satellite *Satellite) getISLInterfaceNames() []string {
	var satelliteNames []string

	for islName := range satellite.ISLInterfaces {
		satelliteNames = append(satelliteNames, islName)
	}
	return satelliteNames
}

func (satellite *Satellite) AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet,
	sendChannel *chan connections.Packet) bool {
	newISLInterface := satellite.ISLInterfaceSample.Clone()
	newISLInterface.SetInterfaceId(id)
	newISLInterface.ChangeSendLink(connectedDevice, sendChannel)
	newISLInterface.ChangeReceiveLink(connectedDevice, receiveChannel)
	satellite.ISLInterfaces[connectedDevice] = newISLInterface
	return true
}

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Name)
	go startSatellite(satellite)
}

func (satellite *Satellite) ProcessBuffers() {
	for _, inface := range satellite.ISLInterfaces {
		if inface.HasSendChannel() {
			inface.ProcessBuffer()
		}
	}
	for _, inface := range satellite.GSLInterfaces {
		if inface.HasSendChannel() {
			inface.ProcessBuffer()
		}
	}
}

func (satellite *Satellite) CheckIncomingConnections() {
	channelEmpty := false
	for !channelEmpty {
		select {
		case linkReq := <-*satellite.LinkerIncomingChannel:
			satellite.ProcessIncomingConnection(linkReq)
		default:
			channelEmpty = true
		}
	}
}

func (satellite *Satellite) ProcessIncomingConnection(linkReq LinkRequest) {
	if satellite.Orbit.IsOwnerSatellite(linkReq.FromDevice) {
		inface, found := satellite.ISLInterfaces[linkReq.FromDevice]
		if found {
			inface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
		} else {
			newInterface := satellite.ISLInterfaceSample.Clone()
			newInterface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
			satellite.ISLInterfaces[linkReq.FromDevice] = newInterface
		}
	} else {
		inface, found := satellite.GSLInterfaces[linkReq.FromDevice]
		if found {
			inface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
		} else {
			newInterface := satellite.GSLInterfaceSample.Clone()
			newInterface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
			satellite.GSLInterfaces[linkReq.FromDevice] = newInterface
		}
	}
}

func (satellite *Satellite) SendPendingRequests() {
	indx := 0
	for indx < len(satellite.PendingConnections) {
		select {
		case *satellite.LinkerOutgoingChannel <- satellite.PendingConnections[indx]:
			satellite.PendingConnections = append(satellite.PendingConnections[:indx], satellite.PendingConnections[indx+1:]...)
		default:
			indx++
		}
	}
}

func (satellite *Satellite) ReceiveFromInterfaces() {
	aliveInterfaces := make(map[string]connections.INetworkInterface)
	for connectedDevice, inface := range satellite.ISLInterfaces {
		if inface.GetDeviceConnectedTo() != "" {
			if inface.HasReceiveChannel() {
				receivedEvents := inface.Receive()
				for _, event := range receivedEvents {
					item := connections.Item{Value: &event, Rank: int(event.TimeStamp)}
					heap.Push(&satellite.EventQueue, &item)
					satellite.logEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, inface.GetDeviceConnectedTo(), satellite.Name)
				}
			}
			aliveInterfaces[connectedDevice] = inface
		}
	}
	satellite.ISLInterfaces = aliveInterfaces
	for gsName, inface := range satellite.GSLInterfaces {
		if inface.GetDeviceConnectedTo() != "" {
			if inface.HasReceiveChannel() {
				receivedEvents := inface.Receive()
				for _, event := range receivedEvents {
					item := connections.Item{Value: &event, Rank: int(event.TimeStamp)}
					heap.Push(&satellite.EventQueue, &item)
					satellite.logEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, inface.GetDeviceConnectedTo(), satellite.Name)
				}
			}
		} else {
			delete(satellite.GSLInterfaces, gsName)
		}
	}
}

func (satellite *Satellite) logEvent(timeStamp int, eventType int, packet *connections.Packet, srcDevice string, destDevice string) {
	*satellite.LoggerChannel <- SimulationEvent{
		TimeStamp:  timeStamp,
		EventType:  eventType,
		FromDevice: srcDevice,
		ToDevice:   destDevice,
		Packet:     packet,
	}
}

func (satellite *Satellite) findISLConnection(toSatellite string) connections.INetworkInterface {
	inface, ok := satellite.ISLInterfaces[toSatellite]
	if ok {
		if inface.HasSendChannel() {
			return inface
		} else {
			satellite.establishSendChannel(inface, toSatellite)
			return inface
		}
	} else {
		return satellite.establishISLConnection(toSatellite)
	}
}

func (satellite *Satellite) findGSLConnection(toGroundStation string) connections.INetworkInterface {
	inface, ok := satellite.GSLInterfaces[toGroundStation]
	if ok {
		if inface.HasSendChannel() {
			return inface
		} else {
			satellite.establishSendChannel(inface, toGroundStation)
			return inface
		}
	} else {
		return satellite.establishGSLConnection(toGroundStation)
	}
}

func (satellite *Satellite) establishSendChannel(inface connections.INetworkInterface, ToDevice string) {
	sendChannel := make(chan connections.Packet, satellite.InterfaceBufferSize)
	inface.ChangeSendLink(inface.GetDeviceConnectedTo(), &sendChannel)
	linkRequest := LinkRequest{
		ToDevice:    ToDevice,
		FromDevice:  satellite.Name,
		SendChannel: &sendChannel,
	}
	select {
	case *satellite.LinkerOutgoingChannel <- linkRequest:
		return
	default:
		satellite.PendingConnections = append(satellite.PendingConnections, linkRequest)
	}
}

func (satellite *Satellite) establishGSLConnection(toGroundStation string) connections.INetworkInterface {
	newNetworkInterface := satellite.GSLInterfaceSample.Clone()
	satellite.GSLInterfaces[toGroundStation] = newNetworkInterface
	sendChannel := make(chan connections.Packet, satellite.InterfaceBufferSize)
	newNetworkInterface.ChangeSendLink(toGroundStation, &sendChannel)
	linkRequest := LinkRequest{
		ToDevice:    toGroundStation,
		FromDevice:  satellite.Name,
		SendChannel: &sendChannel,
	}
	select {
	case *satellite.LinkerOutgoingChannel <- linkRequest:
		return newNetworkInterface
	default:
		satellite.PendingConnections = append(satellite.PendingConnections, linkRequest)
	}

	return newNetworkInterface
}

func (satellite *Satellite) establishISLConnection(toSatellite string) connections.INetworkInterface {
	newNetworkInterface := satellite.ISLInterfaceSample.Clone()
	satellite.ISLInterfaces[toSatellite] = newNetworkInterface
	sendChannel := make(chan connections.Packet, satellite.InterfaceBufferSize)
	newNetworkInterface.ChangeSendLink(toSatellite, &sendChannel)
	linkRequest := LinkRequest{
		ToDevice:    toSatellite,
		FromDevice:  satellite.Name,
		SendChannel: &sendChannel,
	}
	select {
	case *satellite.LinkerOutgoingChannel <- linkRequest:
		return newNetworkInterface
	default:
		satellite.PendingConnections = append(satellite.PendingConnections, linkRequest)
	}

	return newNetworkInterface
}

func (satellite *Satellite) SendTimeStampAck(nextTimeStamp float64) {
	if satellite.lastAckTimeStamp < satellite.TimeStamp && satellite.TimeStamp < nextTimeStamp {
		*satellite.AckTokenChannel <- AckToken{
			TimeStampAck:  satellite.TimeStamp,
			NextTimeStamp: nextTimeStamp,
		}
		satellite.lastAckTimeStamp = satellite.TimeStamp
	}
}

func (satellite *Satellite) getReceiveGSL() ([]*chan connections.Packet, []string) {
	channels := make([]*chan connections.Packet, 0)
	channelOwners := make([]string, 0)
	for gsName, inface := range satellite.GSLInterfaces {
		if inface.GetDeviceConnectedTo() != "" {
			if inface.HasReceiveChannel() {
				channels = append(channels, inface.GetReceiveChannel())
				channelOwners = append(channelOwners, gsName)
			}
		} else {
			delete(satellite.GSLInterfaces, gsName)
		}
	}
	return channels, channelOwners
}

func (satellite *Satellite) getReceiveISL() ([]*chan connections.Packet, []string) {
	channels := make([]*chan connections.Packet, 0)
	channelOwners := make([]string, 0)
	for connectedDevice, inface := range satellite.ISLInterfaces {
		if inface.GetDeviceConnectedTo() != "" {
			if inface.HasReceiveChannel() {
				channels = append(channels, inface.GetReceiveChannel())
				channelOwners = append(channelOwners, connectedDevice)
			}
		} else {
			delete(satellite.ISLInterfaces, connectedDevice)
		}
	}
	return channels, channelOwners
}

func (satellite *Satellite) InitChannelCases(selectCases *[]reflect.SelectCase) ([]string, []string) {
	channelsGSL, channelGSLOwners := satellite.getReceiveGSL()
	channelsISL, channelISLOwners := satellite.getReceiveISL()
	*selectCases = make([]reflect.SelectCase, len(channelsGSL)+len(channelsISL)+2)
	(*selectCases)[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*satellite.ProgressTokenChannel)}
	(*selectCases)[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*satellite.LinkerIncomingChannel)}
	for i, channel := range channelsGSL {
		(*selectCases)[i+2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
	for i, channel := range channelsISL {
		(*selectCases)[i+len(channelGSLOwners)+2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
	return channelGSLOwners, channelISLOwners
}

func (satellite *Satellite) SendPackets() float64 {
	nextEventTime := satellite.TotalSimulationTime
	for !satellite.EventQueue.IsEmpty() {
		itemPopped := heap.Pop(&satellite.EventQueue).(*connections.Item)
		eventType := itemPopped.Value.Type
		nextEventTime = itemPopped.Value.TimeStamp
		if nextEventTime > satellite.TimeStamp {
			heap.Push(&satellite.EventQueue, itemPopped)
			break
		}
		if eventType == connections.SEND_EVENT {
			packet := *itemPopped.Value.Data
			roundedTimeStamp := int(nextEventTime/satellite.Dt) * int(satellite.Dt)
			forwardingChoice := satellite.ForwardingTable[roundedTimeStamp][packet.Destination]
			if satellite.Orbit.IsOwnerSatellite(forwardingChoice) {
				connection := satellite.findISLConnection(forwardingChoice)
				packetDropped, timeOfAttempt := connection.Send(packet, itemPopped.Value.TimeStamp)
				if !packetDropped {
					satellite.logEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, satellite.Name, connection.GetDeviceConnectedTo())
				} else {
					satellite.logEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, satellite.Name, connection.GetDeviceConnectedTo())
				}
			} else {
				connection := satellite.findGSLConnection(forwardingChoice)
				packetDropped, timeOfAttempt := connection.Send(packet, itemPopped.Value.TimeStamp)
				if !packetDropped {
					satellite.logEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, satellite.Name, connection.GetDeviceConnectedTo())
				} else {
					satellite.logEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, satellite.Name, connection.GetDeviceConnectedTo())
				}
			}
		}
		nextEventTime = satellite.TotalSimulationTime
	}
	return nextEventTime
}

func (satellite *Satellite) WatchEvents() {
	var selectIncomingChannelsCases []reflect.SelectCase
	channelGSLOwners, channelISLOwners := satellite.InitChannelCases(&selectIncomingChannelsCases)
	indx, value, _ := reflect.Select(selectIncomingChannelsCases)
	switch indx {
	case 0:
		token := value.Interface().(ProgressToken)
		satellite.TimeStamp = max(satellite.TimeStamp, token.TimeStamp)
	case 1:
		linkReq := value.Interface().(LinkRequest)
		satellite.ProcessIncomingConnection(linkReq)
	default:
		var inface connections.INetworkInterface
		if indx < (len(channelGSLOwners) + 2) {
			inface = satellite.GSLInterfaces[channelGSLOwners[indx-2]]
		} else {
			inface = satellite.ISLInterfaces[channelISLOwners[indx-len(channelGSLOwners)-2]]
		}
		packet := value.Interface().(connections.Packet)
		event := inface.ProcessReceivedPacket(&packet)
		item := connections.Item{Value: &event, Rank: int(event.TimeStamp)}
		heap.Push(&satellite.EventQueue, &item)
		satellite.logEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, inface.GetDeviceConnectedTo(), satellite.Name)
	}
}

func startSatellite(mySatellite ISatellite) {
	for {
		if mySatellite.IsBlocking() {
			mySatellite.WatchEvents()
		}
		mySatellite.CheckIncomingConnections()
		mySatellite.ReceiveFromInterfaces()
		mySatellite.SendPendingRequests()
		nextTimeStamp := mySatellite.SendPackets()
		mySatellite.ProcessBuffers()
		mySatellite.SendTimeStampAck(nextTimeStamp)
	}
}
