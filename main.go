package main

import (
	"fmt"
	"os"

	"github.com/andrejsstepanovs/prowiki/internal/cli"
)

func main() {
	app := cli.NewApp()
	// Pass command-line arguments (excluding the program name itself)
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
