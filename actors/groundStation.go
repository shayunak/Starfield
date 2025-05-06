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
	GSLInterfaces        connections.INetworkInterface
	DistanceSpaceChannel *DistanceSpaceDeviceChannel
	SpaceChannel         *SpaceSatelliteChannel
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
	SetForwardingTable(forwardingTable map[int]ForwardingEntry)
	GenerateTraffic(traffic []TrafficEntry, maxPacketSize float64) int
	generatePackets(maxPacketSize float64, entry TrafficEntry) []connections.Packet
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
	headPointAnomalyEl helpers.AnomalyElements) IGroundStation {

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

func (gs *GroundStation) Run() {
	log.Default().Println("Running ground station: ", gs.Name)
	go startGS(gs)
}

func startGS(myGS IGroundStation) {

}

func (gs *GroundStation) generatePackets(maxPacketSize float64, entry TrafficEntry) []connections.Packet {
	numberOfFullPackets := int(math.Ceil(1000 * entry.Length / maxPacketSize))
	packets := make([]connections.Packet, numberOfFullPackets)

	for i := 0; i < numberOfFullPackets; i++ {
		packets[i] = connections.Packet{
			Source:         gs.Name,
			Destination:    entry.Destination,
			Length:         maxPacketSize,
			PacketSentTime: float64(entry.TimeStamp),
		}
	}
	return packets
}

func (gs *GroundStation) GenerateTraffic(traffic []TrafficEntry, maxPacketSize float64) int {
	number_of_packets := 0
	for _, entry := range traffic {
		packets := gs.generatePackets(maxPacketSize, entry)
		number_of_packets = len(packets)
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
