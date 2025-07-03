package actors

import (
	"log"
	"slices"

	"github.com/shayunak/SatSimGo/connections"
)

type Linker struct {
	// Simulation Mode
	LinkIncomingRequestChannels *LinkRequestChannels
	LinkRelayRequestChannels    *LinkRequestChannels
	DeviceNames                 []string
	PendingConnections          []LinkRequest
}

type LinkRequest struct {
	FromDevice  string
	ToDevice    string
	SendChannel *chan connections.Packet
}

type LinkRequestChannel chan LinkRequest
type LinkRequestChannels []*LinkRequestChannel

type ILinker interface {
	// Simulation Mode
	SetDeviceChannels(incomingChannels *LinkRequestChannels, outgoingChannels *LinkRequestChannels, deviceNames []string)
	GetNumberOfDevices() int
	GetIncomingRequestChannels() *LinkRequestChannels
	GetOutgoingRequestChannels() *LinkRequestChannels
	sendRequest(request LinkRequest) bool
	addPendingRequests(request []LinkRequest)
	processRequests()
	checkIncomingRequests() []LinkRequest
	Run()
}

func (linker *Linker) GetNumberOfDevices() int {
	return len(linker.DeviceNames)
}

func (linker *Linker) GetIncomingRequestChannels() *LinkRequestChannels {
	return linker.LinkIncomingRequestChannels
}

func (linker *Linker) GetOutgoingRequestChannels() *LinkRequestChannels {
	return linker.LinkRelayRequestChannels
}

func (linker *Linker) SetDeviceChannels(incomingChannels *LinkRequestChannels, outgoingChannels *LinkRequestChannels, deviceNames []string) {
	linker.LinkIncomingRequestChannels = incomingChannels
	linker.LinkRelayRequestChannels = outgoingChannels
	linker.DeviceNames = deviceNames
}

func (linker *Linker) addPendingRequests(requests []LinkRequest) {
	linker.PendingConnections = append(linker.PendingConnections, requests...)
}

func (linker *Linker) checkIncomingRequests() []LinkRequest {
	requests := make([]LinkRequest, 0)
	for _, channel := range *linker.GetIncomingRequestChannels() {
		select {
		case request := <-*channel:
			requests = append(requests, request)
		default:
			continue
		}
	}
	return requests
}

func startLinker(linker ILinker) {
	for {
		requests := linker.checkIncomingRequests()
		linker.addPendingRequests(requests)
		linker.processRequests()
	}
}

func (linker *Linker) sendRequest(request LinkRequest) bool {
	channels := *linker.GetOutgoingRequestChannels()
	destIndex := slices.IndexFunc(linker.DeviceNames, func(name string) bool { return name == request.ToDevice })
	if destIndex == -1 {
		return false
	}
	select {
	case *channels[destIndex] <- request:
		return true
	default:
		return false
	}

}

func (linker *Linker) processRequests() {
	indx := 0
	for indx < len(linker.PendingConnections) {
		if linker.sendRequest(linker.PendingConnections[indx]) {
			linker.PendingConnections = append(linker.PendingConnections[:indx], linker.PendingConnections[indx+1:]...)
		} else {
			indx++
		}
	}
}

func (linker *Linker) Run() {
	log.Default().Println("Running Linker...")
	go startLinker(linker)
}
