package main

import (
	"os"

	"github.com/scheduler0/scheduler0-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

