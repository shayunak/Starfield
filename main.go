package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/shayunak/SatSimGo/setup"
)

func main() {
	args := os.Args
	if len(args) == 2 && args[1] == "--help" {
		fmt.Println("main.go [consellation config file] [time step (ms)] [total simulation time (s)]")
		os.Exit(1)
	}
	if len(args) != 4 {
		fmt.Printf("4 arguments required, recieved %d!\n", len(args)-1)
		os.Exit(1)
	}

	timeStep, error := strconv.Atoi(args[2])
	if error != nil {
		fmt.Printf("2nd argument must be an integer, recieved %s!\n", args[2])
		os.Exit(1)
	}

	totalSimulationTime, error := strconv.Atoi(args[3])
	if error != nil {
		fmt.Printf("3rd argument must be an integer, recieved %s!\n", args[3])
		os.Exit(1)
	}

	simulationDone := new(sync.WaitGroup)
	simulationDone.Add(1)

	setup.SetupSimulator(args[1], timeStep, totalSimulationTime, simulationDone)

	simulationDone.Wait() // waiting for the simulation to finish

	log.Default().Println("Simulation Done...")
}
