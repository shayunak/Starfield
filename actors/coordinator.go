package actors

import (
	"log"
	"reflect"
)

type Coordinator struct {
	// Simulation Mode
	ProgressTokenChannels *ProgressTokenChannels
	AckTokenChannels      *AckTokenChannels
	LoggerChannel         *chan float64
	TimeStamp             float64
	NextTimeStamp         float64
	NumberOfAcksPerRound  int
	TotalSimulationTime   float64 // in milliseconds
}

type ProgressToken struct {
	TimeStamp float64
}

type AckToken struct {
	TimeStampAck  float64
	NextTimeStamp float64
}

type ProgressTokenChannel chan ProgressToken
type ProgressTokenChannels []*ProgressTokenChannel

type AckTokenChannel chan AckToken
type AckTokenChannels []*AckTokenChannel

type ICoordinator interface {
	// Simulation Mode
	SetProgressTokenChannels(channels *ProgressTokenChannels)
	SetAckTokenChannels(channels *AckTokenChannels)
	GetNumberOfDevices() int
	GetTimeStamp() float64
	GetNumberOfAcksPerRound() int
	GetTotalSimulationTime() float64
	InitChannelCases(selectCases *[]reflect.SelectCase)
	InitiateNewRound()
	ProcessAckToken(token AckToken)
	Run()
}

func (coordinator *Coordinator) GetNumberOfDevices() int {
	return len(*coordinator.ProgressTokenChannels)
}

func (coordinator *Coordinator) SetProgressTokenChannels(channels *ProgressTokenChannels) {
	coordinator.ProgressTokenChannels = channels
}

func (coordinator *Coordinator) SetAckTokenChannels(channels *AckTokenChannels) {
	coordinator.AckTokenChannels = channels
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
	channels := *coordinator.AckTokenChannels
	for i, channel := range channels {
		(*selectCases)[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(*channel)}
	}
}

func (coordinator *Coordinator) GetNumberOfAcksPerRound() int {
	return coordinator.NumberOfAcksPerRound
}

func (coordinator *Coordinator) InitiateNewRound() {
	coordinator.NumberOfAcksPerRound = 0
	coordinator.TimeStamp = coordinator.NextTimeStamp
	for _, channel := range *coordinator.ProgressTokenChannels {
		*channel <- ProgressToken{TimeStamp: coordinator.TimeStamp}
	}
	*coordinator.LoggerChannel <- coordinator.TimeStamp
	coordinator.NextTimeStamp = coordinator.TotalSimulationTime
}

func (coordinator *Coordinator) ProcessAckToken(token AckToken) {
	coordinator.NumberOfAcksPerRound++
	coordinator.NextTimeStamp = min(coordinator.NextTimeStamp, token.NextTimeStamp)
}

func startCoordinator(coordinator ICoordinator) {
	coordinator.InitiateNewRound()
	for coordinator.GetTimeStamp() <= coordinator.GetTotalSimulationTime() {
		selectDevicesCases := make([]reflect.SelectCase, coordinator.GetNumberOfDevices())
		coordinator.InitChannelCases(&selectDevicesCases)
		_, value, _ := reflect.Select(selectDevicesCases)
		AckToken := value.Interface().(AckToken)
		coordinator.ProcessAckToken(AckToken)
		if coordinator.GetNumberOfAcksPerRound() == coordinator.GetNumberOfDevices() {
			coordinator.InitiateNewRound()
		}
	}
}

func (coordinator *Coordinator) Run() {
	log.Default().Println("Running Coordinator...")
	go startCoordinator(coordinator)
}
