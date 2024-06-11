package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/shayunak/SatSimGo/setup"
)

func calculateDistancesSettingsRun(consellationFile string, timeStepString string, totalSimulationTimeString string) {
	timeStep, error := strconv.Atoi(timeStepString)
	if error != nil {
		fmt.Printf("2nd argument must be an integer, recieved %s!\n", timeStepString)
		os.Exit(1)
	}

	totalSimulationTime, error := strconv.Atoi(totalSimulationTimeString)
	if error != nil {
		fmt.Printf("3rd argument must be an integer, recieved %s!\n", totalSimulationTimeString)
		os.Exit(1)
	}

	simulationDone := new(sync.WaitGroup)
	simulationDone.Add(1)

	setup.SetupSimulatorDistances(consellationFile, timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish
}

func dijkstraSettingsRun(consellationFile string, forwardingFolder string, timeStepString string, totalSimulationTimeString string) {
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

	setup.SetupSimulatorDijkstraSimulation(consellationFile, forwardingFolder, timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish
}

func main() {
	args := os.Args
	if len(args) == 2 && args[1] == "--help" {
		fmt.Println("main.go --help")
		fmt.Println("main.go --distances [consellation config file] [time step (ms)] [total simulation time (s)]")
		fmt.Println("main.go --dijkstra [consellation config file] [forwarding folder] [time step (ms)] [total simulation time (s)]")
		os.Exit(1)
	} else if len(args) == 5 && args[1] == "--distances" {
		calculateDistancesSettingsRun(args[2], args[3], args[4])
		log.Default().Println("Distances Generated...")
	} else if len(args) == 6 && args[1] == "--dijkstra" {
		dijkstraSettingsRun(args[2], args[3], args[4], args[5])
		log.Default().Println("Simulation Done...")
	} else {
		fmt.Println("Invalid Option!")
		os.Exit(1)
	}
}
