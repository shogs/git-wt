package main

import (
	"os"

	"github.com/shogs/git-wt/cmd"
)

// version is set via ldflags by GoReleaser
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
