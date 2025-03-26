package actors

import (
	"container/heap"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/shayunak/SatSimGo/connections"
	"github.com/shayunak/SatSimGo/helpers"
	"github.com/shayunak/SatSimGo/routing"
)

type TrafficEntry struct {
	Destination string
	TimeStamp   int // in milliseconds
	Length      int // in Mb
}

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
	ForwardingFile  string
	ForwardingTable map[int]ForwardingEntry
	EventQueue      helpers.PriorityQueue

	// Goroutines and connections, and channels
	ISLInterfaces        []connections.INetworkInterface
	GSLInterfaces        []connections.INetworkInterface
	AvailableISL         int
	AvailableGSL         int
	DistanceSpaceChannel *DistanceSpaceDeviceChannel
	SpaceChannel         *SpaceSatelliteChannel
}

type ISatellite interface {
	Run()
	RunDistances()
	SetForwardingFile(folder string)
	GetSpaceChannel() *SpaceSatelliteChannel
	SetSpaceChannel(channel *SpaceSatelliteChannel)
	GetDistanceSpaceChannel() *DistanceSpaceDeviceChannel
	SetDistanceSpaceChannel(channel *DistanceSpaceDeviceChannel)
	GetName() string
	GenerateTraffic(traffic []TrafficEntry, maxPacketSize int)
	AddISLConnection(connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	AddISLConnectionOnId(id int, connectedDevice string, receiveChannel *chan connections.Packet, sendChannel *chan connections.Packet) bool
	ReceiveFromInterfaces()
	SendPackets()
	findSatellitesInRange() map[string]helpers.DistanceObject
	findGroundStationsInRange() map[string]helpers.DistanceObject
	findAvailableISLInterfaceId() int
	generatePackets(maxPacketSize int, entry TrafficEntry) []connections.Packet
	getTimeStamp() int
	getTotalSimulationTime() int
	getISLInterfaceNames() []string
	updatePosition()
	updateSpaceOnDistances()
	nextTimeStep()
	loadForwardingTableInMemory()
	updateLinks(distances map[string]float64)
}

func (satellite *Satellite) SetForwardingFile(folder string) {
	satellite.ForwardingFile = fmt.Sprintf("./forwarding_table/%s/%s.csv", folder, satellite.Name)
}

func (satellite *Satellite) GetName() string {
	return satellite.Name
}

func (satellite *Satellite) findSatellitesInRange() map[string]helpers.DistanceObject {
	satelliteOrbitalAscension := satellite.Orbit.GetAscension()
	lengthLimitRatio := satellite.AnomalyCalculations.GetLengthLimitRatio()
	return satellite.AnomalyCalculations.FindSatellitesInRange(satellite.Name, lengthLimitRatio, satellite.OrbitalAnomaly, satellite.AnomalyElements,
		satelliteOrbitalAscension, float64(satellite.TimeStamp)*0.001)
}

func (satellite *Satellite) findGroundStationsInRange() map[string]helpers.DistanceObject {
	return satellite.Orbit.GetCoveringGroundStations(float64(satellite.TimeStamp)*0.001, satellite.OrbitalAnomaly,
		satellite.AnomalyCalculations)
}

func (satellite *Satellite) getTimeStamp() int {
	return satellite.TimeStamp
}

func (satellite *Satellite) getTotalSimulationTime() int {
	return satellite.TotalSimulationTime
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

func (satellite *Satellite) updateLinks(distances map[string]float64) {
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

func (satellite *Satellite) generatePackets(maxPacketSize int, entry TrafficEntry) []connections.Packet {
	numberOfFullPackets := int(entry.Length / maxPacketSize)
	sizeOfLastPacket := entry.Length % maxPacketSize
	packets := make([]connections.Packet, numberOfFullPackets)

	for i := 0; i < numberOfFullPackets; i++ {
		packets[i] = connections.Packet{
			Source:         satellite.Name,
			Destination:    entry.Destination,
			Length:         maxPacketSize,
			PacketSentTime: float64(entry.TimeStamp),
		}
	}
	if sizeOfLastPacket > 0 {
		packets = append(packets, connections.Packet{
			Source:         satellite.Name,
			Destination:    entry.Destination,
			Length:         sizeOfLastPacket,
			PacketSentTime: float64(entry.TimeStamp),
		})
	}
	return packets
}

func (satellite *Satellite) GenerateTraffic(traffic []TrafficEntry, maxPacketSize int) {
	for _, entry := range traffic {
		packets := satellite.generatePackets(maxPacketSize, entry)
		for index, packet := range packets {
			event := connections.Event{
				TimeStamp: float64(entry.TimeStamp),
				Type:      connections.SEND_EVENT,
				Data:      &packet,
			}
			item := helpers.Item{
				Value: &event,
				Rank:  entry.TimeStamp,
				Index: index,
			}
			satellite.EventQueue = append(satellite.EventQueue, &item)
		}
	}

	heap.Init(&satellite.EventQueue)
}

func (satellite *Satellite) RunDistances() {
	log.Default().Println("Running satellite (Distance Mode): ", satellite.Id)
	go startSatelliteDistances(satellite)
}

func (satellite *Satellite) Run() {
	log.Default().Println("Running satellite: ", satellite.Id)
	go startSatellite(satellite)
}

func (satellite *Satellite) loadForwardingTableInMemory() {
	satellite.ForwardingTable = make(map[int]ForwardingEntry)

	file, err := os.Open(satellite.ForwardingFile)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)

	// ignore the header
	_, _ = reader.Read()
	// read the data
	records, _ := reader.ReadAll()

	for _, record := range records {
		timeStamp, _ := strconv.Atoi(record[0])
		if satellite.ForwardingTable[timeStamp] == nil {
			satellite.ForwardingTable[timeStamp] = make(ForwardingEntry)
		}
		satellite.ForwardingTable[timeStamp][record[1]] = record[2]
	}
}

func (satellite *Satellite) GetSpaceChannel() *SpaceSatelliteChannel {
	return satellite.SpaceChannel
}

func (satellite *Satellite) SetSpaceChannel(channel *SpaceSatelliteChannel) {
	satellite.SpaceChannel = channel
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

func mergeMaps(satelliteMap map[string]helpers.DistanceObject, groundStationMap map[string]helpers.DistanceObject) map[string]helpers.DistanceObject {
	mergedMap := make(map[string]helpers.DistanceObject)

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
	newSatellite.EventQueue = make(helpers.PriorityQueue, 0)
	newSatellite.AvailableISL = numberOfIsls
	newSatellite.AvailableGSL = numberOfGsls
	newSatellite.ISLInterfaces = connections.InitISLs(numberOfIsls, speedOfLightVac, bandwidth, linkNoiseCoefficient)

	return &newSatellite
}

func startSatelliteDistances(mySatellite ISatellite) {
	for mySatellite.getTimeStamp() <= mySatellite.getTotalSimulationTime() {
		mySatellite.updateSpaceOnDistances()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
	close(*mySatellite.GetDistanceSpaceChannel())
}

func startSatellite(mySatellite ISatellite) {
	mySatellite.loadForwardingTableInMemory()
	for mySatellite.getTimeStamp() <= mySatellite.getTotalSimulationTime() {
		//satellitesInRange := mySatellite.findSatellitesInRange()
		//mySatellite.updateLinks(satellitesInRange)
		mySatellite.ReceiveFromInterfaces()
		mySatellite.SendPackets()
		mySatellite.nextTimeStep()
		mySatellite.updatePosition()
	}
}

func (satellite *Satellite) ReceiveFromInterfaces() {
	for _, inteface := range satellite.ISLInterfaces {
		if inteface.GetDeviceConnectedTo() != "" {
			receivedEvents := inteface.Receive(float64(satellite.getTimeStamp()))
			for _, event := range receivedEvents {
				heap.Push(&satellite.EventQueue, event)
			}
		}
	}
}

func (satellite *Satellite) SendPackets() {
	lastTimeStampSent := 0.0
	for lastTimeStampSent <= float64(satellite.getTimeStamp()) && !satellite.EventQueue.IsEmpty() {
		itemPopped := heap.Pop(&satellite.EventQueue).(*helpers.Item)
		lastTimeStampSent = itemPopped.Value.TimeStamp
		eventType := itemPopped.Value.Type
		if eventType == connections.SEND_EVENT {
			packet := *itemPopped.Value.Data
			forwardingChoice := satellite.ForwardingTable[satellite.TimeStamp][packet.Destination]
			interfaceId := routing.DijkstraModifiedOnGridPlus(forwardingChoice, satellite.getTimeStamp(), satellite.getISLInterfaceNames(), satellite.AnomalyCalculations)
			if interfaceId != -1 {
				satellite.ISLInterfaces[interfaceId].Send(packet, lastTimeStampSent)
			} else {
				heap.Push(&satellite.EventQueue, itemPopped)
			}
		}
	}
}
