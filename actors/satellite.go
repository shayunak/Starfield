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
	AnomalyElements     helpers.AnomalyElements
	Orbit               helpers.IOrbit
	OrbitalAnomaly      float64 // in radians
	AnomalyCalculations helpers.IAnomalyCalculation

	// Packet Level Simulation
	ForwardingTable map[int]ForwardingEntry
	EventQueue      connections.PriorityQueue

	// Goroutines and connections, and channels
	ISLInterfaces        []connections.INetworkInterface
	GSLInterfaces        []connections.INetworkInterface
	AvailableISL         int
	AvailableGSL         int
	DistanceSpaceChannel *DistanceSpaceDeviceChannel
	SpaceChannel         *SpaceSatelliteChannel
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
	GetSpaceChannel() *SpaceSatelliteChannel
	SetSpaceChannel(channel *SpaceSatelliteChannel)
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	AddISLConnection(connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	ReceiveFromInterfaces()
	SendPackets()
	findAvailableISLInterfaceId() int
	getISLInterfaceNames() []string
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
	anomalyCalculations helpers.IAnomalyCalculation, numberOfIsls int, numberOfGsls int, speedOfLightVac float64,
	bandwidth float64, linkNoiseCoefficient float64, acquisitionTime float64) ISatellite {
	var newSatellite Satellite

	newSatellite.Id = id
	newSatellite.Name = fmt.Sprintf("%s-%d", orbit.GetOrbitId(), id)
	newSatellite.Dt = dt
	newSatellite.TotalSimulationTime = totalSimulationTime
	newSatellite.TimeStamp = 0
	// Geo
	newSatellite.OrbitalAnomaly = orbitalPhase * (math.Pi / 180.0)
	newSatellite.AnomalyCalculations = anomalyCalculations
	newSatellite.Orbit = orbit
	newSatellite.AnomalyElements = helpers.AnomalyElements{
		AnomalySinus:   math.Sin(newSatellite.OrbitalAnomaly),
		AnomalyCosinus: math.Cos(newSatellite.OrbitalAnomaly),
	}
	// Channels
	newSatellite.EventQueue = make(connections.PriorityQueue, 0)
	heap.Init(&newSatellite.EventQueue)
	newSatellite.AvailableISL = numberOfIsls
	newSatellite.AvailableGSL = numberOfGsls
	newSatellite.ISLInterfaces = connections.InitISLs(newSatellite.Name, numberOfIsls, speedOfLightVac, bandwidth, linkNoiseCoefficient, anomalyCalculations)

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

func (satellite *Satellite) GetSpaceChannel() *SpaceSatelliteChannel {
	return satellite.SpaceChannel
}

func (satellite *Satellite) SetSpaceChannel(channel *SpaceSatelliteChannel) {
	satellite.SpaceChannel = channel
}

func (satellite *Satellite) findSatellitesInRange() map[string]float64 {
	satelliteOrbitalAscension := satellite.Orbit.GetAscension()
	lengthLimitRatio := satellite.AnomalyCalculations.GetLengthLimitRatio()
	return satellite.AnomalyCalculations.FindSatellitesInRange(satellite.Name, lengthLimitRatio, satellite.AnomalyElements,
		satelliteOrbitalAscension, float64(satellite.TimeStamp)*0.001)
}

func (satellite *Satellite) findGroundStationsInRange() map[string]float64 {
	return satellite.Orbit.GetCoveringGroundStations(float64(satellite.TimeStamp)*0.001, satellite.OrbitalAnomaly,
		satellite.AnomalyCalculations)
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
	for {
		mySatellite.ReceiveFromInterfaces()
		mySatellite.SendPackets()
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
}

func (satellite *Satellite) sendEvent(timeStamp int, eventType int, packet *connections.Packet, srcSatellite string, destSatellite string) {
	*satellite.SpaceChannel <- SimulationEvent{
		TimeStamp:     timeStamp,
		EventType:     eventType,
		FromSatellite: srcSatellite,
		ToSatellite:   destSatellite,
		Packet:        packet,
	}
}

func (satellite *Satellite) SendPackets() {
	lastTimeStampSent := 0.0
	for !satellite.EventQueue.IsEmpty() {
		itemPopped := heap.Pop(&satellite.EventQueue).(*connections.Item)
		lastTimeStampSent = itemPopped.Value.TimeStamp
		eventType := itemPopped.Value.Type
		if eventType == connections.SEND_EVENT {
			packet := *itemPopped.Value.Data
			forwardingChoice := satellite.ForwardingTable[satellite.TimeStamp][packet.Destination]
			interfaceId := routing.DijkstraModifiedOnGridPlus(forwardingChoice, satellite.getTimeStamp(), satellite.getISLInterfaceNames(), satellite.AnomalyCalculations)
			if interfaceId != -1 {
				success, timeOfAttempt := satellite.ISLInterfaces[interfaceId].Send(packet, lastTimeStampSent)
				if success {
					satellite.sendEvent(timeOfAttempt, SIMULATION_EVENT_SENT, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
				} else {
					satellite.sendEvent(timeOfAttempt, SIMULATION_EVENT_DROPPED, &packet, satellite.Name, satellite.ISLInterfaces[interfaceId].GetDeviceConnectedTo())
				}
			} else {
				heap.Push(&satellite.EventQueue, itemPopped)
			}
		}
	}
}
