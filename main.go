package main

import (
	"os"

	"github.com/shogs/git-wt/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
