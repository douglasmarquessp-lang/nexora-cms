package main

import (
	"fmt"
	"os"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

// Pending implementation.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg)
	log.Info("worker starting", "version", "0.1.0")

	select {}
}
