package actors

import (
	"log"
	"reflect"
	"slices"

	"SatSimGo/connections"
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
	InitChannelCases(selectCases *[]reflect.SelectCase)
	sendRequest(request LinkRequest) bool
	addPendingRequest(request LinkRequest)
	processRequests()
	isPendingConnectionsEmpty() bool
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

func (linker *Linker) isPendingConnectionsEmpty() bool {
	return len(linker.PendingConnections) == 0
}

func (linker *Linker) SetDeviceChannels(incomingChannels *LinkRequestChannels, outgoingChannels *LinkRequestChannels, deviceNames []string) {
	linker.LinkIncomingRequestChannels = incomingChannels
	linker.LinkRelayRequestChannels = outgoingChannels
	linker.DeviceNames = deviceNames
}

func (linker *Linker) addPendingRequest(request LinkRequest) {
	linker.PendingConnections = append(linker.PendingConnections, request)
}

func (linker *Linker) InitChannelCases(selectCases *[]reflect.SelectCase) {
	channels := *linker.GetIncomingRequestChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func startLinker(linker ILinker) {
	for {
		if linker.isPendingConnectionsEmpty() {
			selectDevicesCases := make([]reflect.SelectCase, linker.GetNumberOfDevices())
			linker.InitChannelCases(&selectDevicesCases)
			_, value, _ := reflect.Select(selectDevicesCases)
			request := value.Interface().(LinkRequest)
			linker.addPendingRequest(request)
		}
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
