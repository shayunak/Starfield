package actors

import (
	"log"
	"reflect"
	"slices"

	"github.com/shayunak/SatSimGo/connections"
)

type Linker struct {
	// Simulation Mode
	LinkChannels       *LinkRequestChannels
	DeviceNames        []string
	PendingConnections []LinkRequest
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
	SetDeviceChannels(channels *LinkRequestChannels, deviceNames []string)
	GetNumberOfDevices() int
	GetRequestChannels() *LinkRequestChannels
	sendRequest(request LinkRequest) bool
	addPendingRequest(request LinkRequest)
	processRequests()
	Run()
}

func (linker *Linker) GetRequestChannels() *LinkRequestChannels {
	return linker.LinkChannels
}

func (linker *Linker) GetNumberOfDevices() int {
	return len(*linker.LinkChannels)
}

func (linker *Linker) SetDeviceChannels(channels *LinkRequestChannels, deviceNames []string) {
	linker.LinkChannels = channels
	linker.DeviceNames = deviceNames
}

func initRequestChannelsCases(selectCases *[]reflect.SelectCase, linker ILinker) {
	channels := *linker.GetRequestChannels()
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (linker *Linker) addPendingRequest(request LinkRequest) {
	linker.PendingConnections = append(linker.PendingConnections, request)
}

func startLinker(linker ILinker) {
	for {
		selectDevicesCases := make([]reflect.SelectCase, linker.GetNumberOfDevices())
		initRequestChannelsCases(&selectDevicesCases, linker)
		_, value, _ := reflect.Select(selectDevicesCases)
		linkReq := value.Interface().(LinkRequest)
		linker.addPendingRequest(linkReq)
		linker.processRequests()
	}
}

func (linker *Linker) sendRequest(request LinkRequest) bool {
	channels := *linker.GetRequestChannels()
	destIndex := slices.IndexFunc(linker.DeviceNames, func(name string) bool { return name == request.ToDevice })
	if destIndex == -1 {
		log.Default().Println("Unknown destination: ", request.ToDevice)
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
