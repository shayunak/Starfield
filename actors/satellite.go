package actors

import (
	"container/heap"
	"fmt"
	"log"
	"math"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
	"github.com/shayunak/SatSimGo/routing"
)

type ForwardingEntry map[string]string

type Satellite struct {
	// General
	Name                string
	Id                  int
	Dt                  float64 // in milliseconds
	TimeStamp           float64 // in milliseconds
	TotalSimulationTime float64 // in milliseconds

	// Geometrical parameters
	// Position            helpers.CartesianCoordinates (Unnecessary for satellite distances calculations)
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
	InterfaceBufferSize   int
	ISLInterfaces         []connections.INetworkInterface
	AvailableISL          int
	GSLInterfaceSample    connections.INetworkInterface
	GSLInterfaces         map[string]connections.INetworkInterface
	LinkerOutgoingChannel *LinkRequestChannel
	LinkerIncomingChannel *LinkRequestChannel
	DistanceLoggerChannel *DistanceLoggerDeviceChannel
	LoggerChannel         *LoggerDeviceChannel
	ProgressTokenChannel  *ProgressTokenChannel
	PendingConnections    []LinkRequest
}

type ISatellite interface {
	// General
	GetName() string
	getTimeStamp() float64
	getTotalSimulationTime() float64
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
	SetLoggerChannel(channel *LoggerDeviceChannel)
	SetProgressTokenChannel(channel *ProgressTokenChannel)
	SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	ReceiveFromInterfaces()
	SendPackets() float64
	CheckIncomingConnection()
	CheckProgressToken()
	SendTimeStampAck(nextTimeStamp float64)
	SendPendingRequests()
	ProcessBuffers()
	AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	findGSLConnection(toGroundStation string) connections.INetworkInterface
	findAvailableISLInterfaceId() int
	getISLInterfaceNames() []string
	establishGSLConnection(toGroundStation string) connections.INetworkInterface
	establishSendChannel(inface connections.INetworkInterface, toGroundStation string)
	logEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string)
}

func (satellite *Satellite) GetName() string {
	return satellite.Name
}

func (satellite *Satellite) getTimeStamp() float64 {
	return satellite.TimeStamp
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
	newSatellite.TimeStamp = 0

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
	newSatellite.lastAckTimeStamp = 0.0

	// Channels
	newSatellite.InterfaceBufferSize = interfaceBufferSize
	newSatellite.EventQueue = make(connections.PriorityQueue, 0)
	heap.Init(&newSatellite.EventQueue)
	newSatellite.AvailableISL = numberOfIsls
	newSatellite.ISLInterfaces = connections.InitISLs(newSatellite.Name, numberOfIsls, speedOfLightVac, ISLBandwidth,
		ISLLinkNoiseCoefficient, anomalyCalculations, maxPacketSize, interfaceBufferSize)
	newSatellite.GSLInterfaceSample = connections.InitGSL(newSatellite.Name, speedOfLightVac, GSLBandwidth, GSLLinkNoiseCoefficient, orbit,
		newSatellite.OrbitalAnomaly, 0.0, newSatellite.AnomalyElements, groundStationCalculations, maxPacketSize, interfaceBufferSize)
	newSatellite.GSLInterfaces = make(map[string]connections.INetworkInterface)
	newSatellite.PendingConnections = make([]LinkRequest, 0)

	return &newSatellite
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

func (satellite *Satellite) RunDistances() {
	log.Default().Println("Running satellite (Distance Mode): ", satellite.Id)
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
	satellite.TimeStamp += satellite.Dt
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
		satelliteOrbitalAscension, float64(satellite.TimeStamp)*0.001)
}

func (satellite *Satellite) findGroundStationsInRange() map[string]float64 {
	return satellite.GroundStationCalculations.GetCoveringGroundStations(float64(satellite.TimeStamp)*0.001, satellite.OrbitalAnomaly,
		satellite.Orbit)
}

func (satellite *Satellite) logDistances() {
	satelliteDistances := satellite.findSatellitesInRange()
	groundStationDistances := satellite.findGroundStationsInRange()

	(*satellite.DistanceLoggerChannel) <- UpdateDistancesMessage{
		DeviceName: satellite.Name,
		TimeStamp:  int(satellite.TimeStamp),
		Distances:  mergeMaps(satelliteDistances, groundStationDistances),
	}
}

func startSatelliteDistances(mySatellite ISatellite) {
	for mySatellite.getTimeStamp() <= mySatellite.getTotalSimulationTime() {
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

func (satellite *Satellite) SetLinkerChannels(ingoingChannel *LinkRequestChannel, outgoingChannel *LinkRequestChannel) {
	satellite.LinkerIncomingChannel = ingoingChannel
	satellite.LinkerOutgoingChannel = outgoingChannel
}

func (satellite *Satellite) getISLInterfaceNames() []string {
	satelliteNames := make([]string, len(satellite.ISLInterfaces))

	for i := 0; i < len(satellite.ISLInterfaces); i++ {
		satelliteNames[i] = satellite.ISLInterfaces[i].GetDeviceConnectedTo()
	}
	return satelliteNames
}

func (satellite *Satellite) AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet,
	sendChannel *chan connections.Packet) bool {
	if satellite.AvailableISL <= 0 {
		return false
	}
	println("Adding ISL connection on id: ", id, " for satellite: ", satellite.Name, " to device: ", connectedDevice)
	satellite.ISLInterfaces[id].ChangeSendLink(connectedDevice, sendChannel)
	satellite.ISLInterfaces[id].ChangeReceiveLink(connectedDevice, receiveChannel)
	satellite.AvailableISL--
	return true
}

func (satellite *Satellite) findAvailableISLInterfaceId() int {
	for i := 0; i < len(satellite.ISLInterfaces); i++ {
		if satellite.ISLInterfaces[i].GetDeviceConnectedTo() == "" {
			return i
		}
	}
	return -1
}

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Id)
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

func (satellite *Satellite) CheckIncomingConnection() {
	select {
	case linkReq := <-*satellite.LinkerIncomingChannel:
		inface, found := satellite.GSLInterfaces[linkReq.FromDevice]
		if found {
			inface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
		} else {
			newInterface := satellite.GSLInterfaceSample.Clone()
			newInterface.ChangeReceiveLink(linkReq.FromDevice, linkReq.SendChannel)
			satellite.GSLInterfaces[linkReq.FromDevice] = newInterface
		}
	default:
		return
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
	for indx, inface := range satellite.ISLInterfaces {
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
			satellite.ISLInterfaces = append(satellite.ISLInterfaces[:indx], satellite.ISLInterfaces[indx+1:]...)
		}
	}
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

func (satellite *Satellite) establishSendChannel(inface connections.INetworkInterface, toGroundStation string) {
	sendChannel := make(chan connections.Packet, satellite.InterfaceBufferSize)
	inface.ChangeSendLink(inface.GetDeviceConnectedTo(), &sendChannel)
	linkRequest := LinkRequest{
		ToDevice:    toGroundStation,
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

func (satellite *Satellite) SendTimeStampAck(nextTimeStamp float64) {
	if satellite.lastAckTimeStamp < satellite.TimeStamp && satellite.TimeStamp < nextTimeStamp {
		*satellite.ProgressTokenChannel <- ProgressToken{
			CurrentTimeStamp: satellite.TimeStamp,
			NextTimeStamp:    nextTimeStamp,
		}
		satellite.lastAckTimeStamp = satellite.TimeStamp
	}
}

func (satellite *Satellite) CheckProgressToken() {
	select {
	case token := <-*satellite.ProgressTokenChannel:
		if token.CurrentTimeStamp > satellite.TimeStamp {
			satellite.TimeStamp = token.CurrentTimeStamp
		}
	default:
		return
	}
}

func (satellite *Satellite) SendPackets() float64 {
	nextEventTime := 0.0
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
				interfaceId := routing.DijkstraModifiedOnGridPlus(forwardingChoice, satellite.getTimeStamp(), satellite.getISLInterfaceNames(), satellite.AnomalyCalculations)
				if interfaceId != -1 {
					packetDropped, timeOfAttempt := satellite.ISLInterfaces[interfaceId].Send(packet, itemPopped.Value.TimeStamp)
					if !packetDropped {
						satellite.logEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
					} else {
						satellite.logEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
					}
				} else {
					satellite.logEvent(int(nextEventTime), SIMULATION_EVENT_DROPPED, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
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
	}
	return nextEventTime
}

func startSatellite(mySatellite ISatellite) {
	for {
		mySatellite.CheckProgressToken()
		mySatellite.CheckIncomingConnection()
		mySatellite.ReceiveFromInterfaces()
		mySatellite.SendPendingRequests()
		nextTimeStamp := mySatellite.SendPackets()
		mySatellite.ProcessBuffers()
		mySatellite.SendTimeStampAck(nextTimeStamp)
	}
}
