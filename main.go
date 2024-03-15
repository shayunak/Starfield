package main

import (
	"fmt"
	"os"

	"github.com/shayunak/SatSimGo/setup"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Printf("One argument required, recieved %d!\n", len(args)-1)
		os.Exit(1)
	}

	setup.SetupSimulator(args[1])
}
