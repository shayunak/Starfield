package actors

import (
	"log"
	"reflect"
)

type Coordinator struct {
	// Simulation Mode
	ProgressTokenChannels *ProgressTokenChannels
	LoggerChannel         *chan float64
	TimeStamp             float64
	NumberOfAcksPerRound  int
	TotalSimulationTime   float64 // in milliseconds
}

type ProgressToken struct {
	CurrentTimeStamp float64
	NextTimeStamp    float64
}

type ProgressTokenChannel chan ProgressToken
type ProgressTokenChannels []*ProgressTokenChannel

type ICoordinator interface {
	// Simulation Mode
	SetProgressTokenChannels(channels *ProgressTokenChannels)
	GetNumberOfDevices() int
	GetTimeStamp() float64
	GetNumberOfAcksPerRound() int
	GetTotalSimulationTime() float64
	InitChannelCases(selectCases *[]reflect.SelectCase)
	InitiateNewRound()
	ProcessToken(token ProgressToken)
	Run()
}

func (coordinator *Coordinator) GetNumberOfDevices() int {
	return len(*coordinator.ProgressTokenChannels)
}

func (coordinator *Coordinator) SetProgressTokenChannels(channels *ProgressTokenChannels) {
	coordinator.ProgressTokenChannels = channels
}

func (coordinator *Coordinator) SetLoggerChannel(channel *chan float64) {
	coordinator.LoggerChannel = channel
}

func (coordinator *Coordinator) GetTimeStamp() float64 {
	return coordinator.TimeStamp
}

func (coordinator *Coordinator) GetTotalSimulationTime() float64 {
	return coordinator.TotalSimulationTime
}

func (coordinator *Coordinator) InitChannelCases(selectCases *[]reflect.SelectCase) {
	channels := *coordinator.ProgressTokenChannels
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (coordinator *Coordinator) GetNumberOfAcksPerRound() int {
	return coordinator.NumberOfAcksPerRound
}

func (coordinator *Coordinator) InitiateNewRound() {
	coordinator.NumberOfAcksPerRound = 0
	for _, channel := range *coordinator.ProgressTokenChannels {
		*channel <- ProgressToken{CurrentTimeStamp: coordinator.TimeStamp, NextTimeStamp: 0}
	}
	*coordinator.LoggerChannel <- coordinator.TimeStamp // Notify logger about the new round
	coordinator.TimeStamp = coordinator.TotalSimulationTime
}

func (coordinator *Coordinator) ProcessToken(token ProgressToken) {
	coordinator.NumberOfAcksPerRound++
	coordinator.TimeStamp = min(coordinator.TimeStamp, token.NextTimeStamp)
}

func startCoordinator(coordinator ICoordinator) {
	coordinator.InitiateNewRound()
	for coordinator.GetTimeStamp() < coordinator.GetTotalSimulationTime() {
		selectDevicesCases := make([]reflect.SelectCase, coordinator.GetNumberOfDevices())
		coordinator.InitChannelCases(&selectDevicesCases)
		_, value, _ := reflect.Select(selectDevicesCases)
		progressToken := value.Interface().(ProgressToken)
		coordinator.ProcessToken(progressToken)
		if coordinator.GetNumberOfAcksPerRound() == coordinator.GetNumberOfDevices() {
			coordinator.InitiateNewRound()
		}
	}
}

func (coordinator *Coordinator) Run() {
	log.Default().Println("Running Coordinator...")
	go startCoordinator(coordinator)
}
