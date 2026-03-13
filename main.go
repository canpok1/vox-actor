package main

import (
	"os"

	"github.com/canpok1/vox-actor/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
