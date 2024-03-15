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
	fmt.Println(setup.GetConfig(args[1]).ToString())
}
