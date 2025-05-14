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
	Dt                  int // in milliseconds
	TimeStamp           int // in milliseconds
	TotalSimulationTime int // in milliseconds

	// Geometrical parameters
	// Position            helpers.CartesianCoordinates (Unnecessary for satellite distances calculations)
	AnomalyElements           helpers.AnomalyElements
	Orbit                     helpers.IOrbit
	OrbitalAnomaly            float64 // in radians
	AnomalyCalculations       helpers.IAnomalyCalculation
	GroundStationCalculations helpers.IGroundStationCalculation

	// Packet Level Simulation
	ForwardingTable map[int]ForwardingEntry
	EventQueue      connections.PriorityQueue

	// Goroutines and connections, and channels
	ISLInterfaces        []connections.INetworkInterface
	AvailableISL         int
	GSLInterface         connections.INetworkInterface
	DistanceSpaceChannel *DistanceSpaceDeviceChannel
	SpaceChannel         *SpaceDeviceChannel
}

type ISatellite interface {
	// General
	GetName() string
	getTimeStamp() int
	getTotalSimulationTime() int
	// Distance Mode
	RunDistances()
	GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel
	SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel)
	updatePosition()
	updateSpaceOnDistances()
	nextTimeStep()
	findSatellitesInRange() map[string]float64
	findGroundStationsInRange() map[string]float64
	// Simulation Mode
	Run()
	GetSpaceChannel() *SpaceDeviceChannel
	SetSpaceChannel(channel *SpaceDeviceChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	AddISLConnection(connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	ReceiveFromInterfaces()
	SendPackets()
	CheckIncomingConnections() bool
	findAvailableISLInterfaceId() int
	getISLInterfaceNames() []string
	establishConnection(toGroundStation string, timeStamp int)
	sendEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string)
}

func (satellite *Satellite) GetName() string {
	return satellite.Name
}

func (satellite *Satellite) getTimeStamp() int {
	return satellite.TimeStamp
}

func (satellite *Satellite) getTotalSimulationTime() int {
	return satellite.TotalSimulationTime
}

func NewSatellite(id int, orbitalPhase float64, dt int, totalSimulationTime int, orbit helpers.IOrbit,
	anomalyCalculations helpers.IAnomalyCalculation, groundStationCalculations helpers.IGroundStationCalculation,
	numberOfIsls int, speedOfLightVac float64, ISLBandwidth float64, ISLLinkNoiseCoefficient float64,
	GSLBandwidth float64, GSLLinkNoiseCoefficient float64, acquisitionTime float64) ISatellite {
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
	// Channels
	newSatellite.EventQueue = make(connections.PriorityQueue, 0)
	heap.Init(&newSatellite.EventQueue)
	newSatellite.AvailableISL = numberOfIsls
	newSatellite.ISLInterfaces = connections.InitISLs(newSatellite.Name, numberOfIsls, speedOfLightVac, ISLBandwidth, ISLLinkNoiseCoefficient, anomalyCalculations)
	newSatellite.GSLInterface = connections.InitGSL(newSatellite.Name, speedOfLightVac, GSLBandwidth, GSLLinkNoiseCoefficient, orbit,
		newSatellite.OrbitalAnomaly, 0.0, newSatellite.AnomalyElements, groundStationCalculations)

	return &newSatellite
}

//////////////////////////////////// ****** Distances Mode ****** //////////////////////////////////////////////////

func (satellite *Satellite) RunDistances() {
	log.Default().Println("Running satellite (Distance Mode): ", satellite.Id)
	go startSatelliteDistances(satellite)
}

func (satellite *Satellite) GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel {
	return satellite.DistanceSpaceChannel
}

func (satellite *Satellite) SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel) {
	satellite.DistanceSpaceChannel = channel
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

func (satellite *Satellite) updateSpaceOnDistances() {
	satelliteDistances := satellite.findSatellitesInRange()
	groundStationDistances := satellite.findGroundStationsInRange()

	(*satellite.DistanceSpaceChannel) <- UpdateDistancesMessage{
		DeviceName: satellite.Name,
		TimeStamp:  satellite.TimeStamp,
		Distances:  mergeMaps(satelliteDistances, groundStationDistances),
	}
}

func startSatelliteDistances(mySatellite ISatellite) {
	for mySatellite.getTimeStamp() <= mySatellite.getTotalSimulationTime() {
		mySatellite.updateSpaceOnDistances()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
	close(*mySatellite.GetDistanceSpaceChannel())
}

//////////////////////////////////// ****** Simulation Mode ****** //////////////////////////////////////////////////

func (satellite *Satellite) SetForwardingTable(forwardingTable map[int]ForwardingEntry) {
	satellite.ForwardingTable = forwardingTable
}

func (satellite *Satellite) GetSpaceChannel() *SpaceDeviceChannel {
	return satellite.SpaceChannel
}

func (satellite *Satellite) SetSpaceChannel(channel *SpaceDeviceChannel) {
	satellite.SpaceChannel = channel
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

func (satellite *Satellite) getISLInterfaceNames() []string {
	satelliteNames := make([]string, len(satellite.ISLInterfaces))

	for i := 0; i < len(satellite.ISLInterfaces); i++ {
		satelliteNames[i] = satellite.ISLInterfaces[i].GetDeviceConnectedTo()
	}
	return satelliteNames
}

func (satellite *Satellite) findAvailableISLInterfaceId() int {
	for i := 0; i < len(satellite.ISLInterfaces); i++ {
		if satellite.ISLInterfaces[i].GetDeviceConnectedTo() == "" {
			return i
		}
	}
	return -1
}

func (satellite *Satellite) AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet,
	sendChannel *chan connections.Packet) bool {
	if satellite.AvailableISL <= 0 {
		return false
	}
	satellite.ISLInterfaces[id].ChangeLink(connectedDevice, sendChannel, receiveChannel)
	satellite.AvailableISL--
	return true
}

func (satellite *Satellite) AddISLConnection(connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool {
	if satellite.AvailableISL <= 0 {
		return false
	}
	interfaceIndex := satellite.findAvailableISLInterfaceId()
	satellite.ISLInterfaces[interfaceIndex].ChangeLink(connectedDevice, sendChannel, receiveChannel)
	satellite.AvailableISL--
	return true
}

/*
Updating happens when sending packets takes place
func (satellite *Satellite) updateLinks() {
	for i := 0; i < len(satellite.ISLInterfaces); i++ {
		if satellite.ISLInterfaces[i].GetDeviceConnectedTo() != "" {
			distance, ok := distances[satellite.ISLInterfaces[i].GetDeviceConnectedTo()]
			// satellite out of range
			if !ok {
				satellite.ISLInterfaces[i].CloseConnection()
				satellite.AvailableISL++
			} else {
				satellite.ISLInterfaces[i].GetLink().UpdateLink(distance)
			}
		}
	}
}
*/

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Id)
	go startSatellite(satellite)
}

func startSatellite(mySatellite ISatellite) {
	simulationDone := false
	for !simulationDone {
		simulationDone = mySatellite.CheckIncomingConnections()
		mySatellite.ReceiveFromInterfaces()
		mySatellite.SendPackets()
	}
}

func (satellite *Satellite) CheckIncomingConnections() bool {
	select {
	case event, ok := <-*satellite.SpaceChannel:
		if !ok {
			return true
		}
		if event.EventType == SIMULATION_EVENT_CONNECTION_ESTABLISHED {
			satellite.GSLInterface.ChangeLink(event.ToDevice, event.LinkReq.SendChannel, event.LinkReq.RecieveChannel)
		}
		return false
	default:
		return false
	}
}

func (satellite *Satellite) ReceiveFromInterfaces() {
	for _, inface := range satellite.ISLInterfaces {
		if !inface.GetLinkStatus() {
			receivedEvents := inface.Receive()
			for _, event := range receivedEvents {
				heap.Push(&satellite.EventQueue, event)
				satellite.sendEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, inface.GetDeviceConnectedTo(), satellite.Name)
			}
		}
	}
	if !satellite.GSLInterface.GetLinkStatus() {
		receivedEvents := satellite.GSLInterface.Receive()
		for _, event := range receivedEvents {
			heap.Push(&satellite.EventQueue, event)
			satellite.sendEvent(int(event.TimeStamp), SIMULATION_EVENT_RECEIVED, event.Data, satellite.GSLInterface.GetDeviceConnectedTo(), satellite.Name)
		}
	}
}

func (satellite *Satellite) sendEvent(timeStamp int, eventType int, packet *connections.Packet, srcDevice string, destDevice string) {
	*satellite.SpaceChannel <- SimulationEvent{
		TimeStamp:  timeStamp,
		EventType:  eventType,
		FromDevice: srcDevice,
		ToDevice:   destDevice,
		Packet:     packet,
	}
}

func (satellite *Satellite) establishConnection(toGroundStation string, timeStamp int) {
	*satellite.SpaceChannel <- SimulationEvent{
		TimeStamp:  timeStamp,
		EventType:  SIMULATION_EVENT_CONNECTION_ESTABLISHED,
		FromDevice: satellite.Name,
		ToDevice:   toGroundStation,
		Packet:     nil,
		LinkReq:    nil,
	}
	connectionEvent := <-*satellite.SpaceChannel
	satellite.GSLInterface.ChangeLink(connectionEvent.ToDevice, connectionEvent.LinkReq.SendChannel, connectionEvent.LinkReq.RecieveChannel)
}

func (satellite *Satellite) SendPackets() {
	for !satellite.EventQueue.IsEmpty() {
		itemPopped := heap.Pop(&satellite.EventQueue).(*connections.Item)
		eventType := itemPopped.Value.Type
		if eventType == connections.SEND_EVENT {
			packet := *itemPopped.Value.Data
			timeStamp := int(itemPopped.Value.TimeStamp / float64(satellite.Dt))
			forwardingChoice := satellite.ForwardingTable[timeStamp][packet.Destination]
			if satellite.Orbit.IsOwnerSatellite(forwardingChoice) {
				interfaceId := routing.DijkstraModifiedOnGridPlus(forwardingChoice, satellite.getTimeStamp(), satellite.getISLInterfaceNames(), satellite.AnomalyCalculations)
				if interfaceId != -1 {
					success, timeOfAttempt := satellite.ISLInterfaces[interfaceId].Send(packet, itemPopped.Value.TimeStamp)
					if success {
						satellite.sendEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
					} else {
						satellite.sendEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
					}
				} else {
					heap.Push(&satellite.EventQueue, itemPopped)
				}
			} else {
				if satellite.GSLInterface.GetDeviceConnectedTo() != forwardingChoice {
					satellite.establishConnection(forwardingChoice, timeStamp)
				}
				success, timeOfAttempt := satellite.GSLInterface.Send(packet, itemPopped.Value.TimeStamp)
				if success {
					satellite.sendEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, satellite.Name, satellite.GSLInterface.GetDeviceConnectedTo())
				} else {
					satellite.sendEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, satellite.Name, satellite.GSLInterface.GetDeviceConnectedTo())
				}
			}
		}
	}
}
