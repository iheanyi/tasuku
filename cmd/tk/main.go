package main

import (
	"os"

	"github.com/iheanyi/tasuku/internal/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
