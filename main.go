package main

import (
	"os"

	"github.com/scheduler0/scheduler0-cli/cmd"
)

var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

