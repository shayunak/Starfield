package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/shayunak/SatSimGo/setup"
)

func calculatePositionsSettingsRun(consellationFile string, timeStepString string, totalSimulationTimeString string) {
	timeStep, error := strconv.Atoi(timeStepString)
	if error != nil {
		fmt.Printf("Second argument must be an integer, recieved %s!\n", timeStepString)
		os.Exit(1)
	}

	totalSimulationTime, error := strconv.Atoi(totalSimulationTimeString)
	if error != nil {
		fmt.Printf("Third argument must be an integer, recieved %s!\n", totalSimulationTimeString)
		os.Exit(1)
	}

	simulationDone := new(sync.WaitGroup)
	simulationDone.Add(1)

	setup.SetupSimulatorPositions(consellationFile, timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish
}

func calculateDistancesSettingsRun(consellationFile string, groundStationFile string, timeStepString string, totalSimulationTimeString string) {
	timeStep, error := strconv.Atoi(timeStepString)
	if error != nil {
		fmt.Printf("Third argument must be an integer, recieved %s!\n", timeStepString)
		os.Exit(1)
	}

	totalSimulationTime, error := strconv.Atoi(totalSimulationTimeString)
	if error != nil {
		fmt.Printf("Fourth argument must be an integer, recieved %s!\n", totalSimulationTimeString)
		os.Exit(1)
	}

	simulationDone := new(sync.WaitGroup)
	simulationDone.Add(1)

	setup.SetupSimulatorDistances(consellationFile, groundStationFile, timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish
}

func forwardingSettingsRun(consellationFile string, groundStationFile string, trafficFile string, forwardingFolder string, ISLTopologyFileName string, timeStepString string, totalSimulationTimeString string) {
	timeStep, error := strconv.Atoi(timeStepString)
	if error != nil {
		fmt.Printf("3rd argument must be an integer, recieved %s!\n", timeStepString)
		os.Exit(1)
	}

	totalSimulationTime, error := strconv.Atoi(totalSimulationTimeString)
	if error != nil {
		fmt.Printf("4th argument must be an integer, recieved %s!\n", totalSimulationTimeString)
		os.Exit(1)
	}

	simulationDone := new(sync.WaitGroup)
	simulationDone.Add(1)

	setup.SetupForwardingSimulation(consellationFile, groundStationFile, trafficFile, forwardingFolder, ISLTopologyFileName, timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish
}

func forwardingSettingsRunGridPlus(consellationFile string, groundStationFile string, trafficFile string, forwardingFolder string, timeStepString string, totalSimulationTimeString string) {
	timeStep, error := strconv.Atoi(timeStepString)
	if error != nil {
		fmt.Printf("3rd argument must be an integer, recieved %s!\n", timeStepString)
		os.Exit(1)
	}

	totalSimulationTime, error := strconv.Atoi(totalSimulationTimeString)
	if error != nil {
		fmt.Printf("4th argument must be an integer, recieved %s!\n", totalSimulationTimeString)
		os.Exit(1)
	}

	simulationDone := new(sync.WaitGroup)
	simulationDone.Add(1)

	setup.SetupForwardingSimulationGridPlus(consellationFile, groundStationFile, trafficFile, forwardingFolder, timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish
}

func printHelp() {
	fmt.Println("main.go --help")
	fmt.Println("main.go --positions [consellation config file] [time step (ms)] [total simulation time (s)]")
	fmt.Println("main.go --distances [consellation config file] [ground station locations] [time step (ms)] [total simulation time (s)]")
	fmt.Println("main.go --forwarding [consellation config file] [ground station locations] [traffic generator file] [forwarding folder] [ISL Topology] [time step (ms)] [total simulation time (s)]")
	fmt.Println("main.go --forwarding --grid_plus [consellation config file] [ground station locations] [traffic generator file] [forwarding folder] [time step (ms)] [total simulation time (s)]")
}

func main() {
	args := os.Args

	if len(args) < 2 {
		fmt.Println("No Option Provided!")
		printHelp()
		os.Exit(1)
	}

	if args[1] == "--help" && len(args) == 2 {
		printHelp()
	} else if args[1] == "--positions" && len(args) == 5 {
		calculatePositionsSettingsRun(args[2], args[3], args[4])
		log.Default().Println("Positions Generated...")
	} else if args[1] == "--distances" && len(args) == 6 {
		calculateDistancesSettingsRun(args[2], args[3], args[4], args[5])
		log.Default().Println("Distances Generated...")
	} else if args[1] == "--forwarding" && args[2] == "--grid_plus" && len(args) == 9 {
		forwardingSettingsRunGridPlus(args[3], args[4], args[5], args[6], args[7], args[8])
		log.Default().Println("Simulation Done...")
	} else if args[1] == "--forwarding" && len(args) == 9 {
		forwardingSettingsRun(args[2], args[3], args[4], args[5], args[6], args[7], args[8])
		log.Default().Println("Simulation Done...")
	} else {
		fmt.Println("Invalid Option or Missing Arguments!")
		printHelp()
		os.Exit(1)
	}
}
