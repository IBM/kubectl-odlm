package main

import (
	"os"

	"github.com/IBM/kubectl-odlm/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
