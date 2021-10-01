package main

import (
	"fmt"
	"os"

	"github.com/smartpricer/consulenv/commands"
)

//
func main() {
	if err := commands.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
