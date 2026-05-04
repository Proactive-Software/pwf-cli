package main

import (
	"os"

	"github.com/Proactive-Software/pwf-cli/internal/commands"
)

func main() {
	if err := commands.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
