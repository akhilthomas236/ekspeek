package main

import (
	"os"

	"ekspeek/pkg/cmd"
)

func main() {
	if err := cmd.NewEKSCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
