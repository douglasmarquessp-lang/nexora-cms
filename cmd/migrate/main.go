package main

import (
	"fmt"
	"log/slog"
	"os"

	"nexora/internal/pkg/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	m, err := migrate.New(
		"file://migrations",
		cfg.Database.DSN(),
	)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			slog.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied successfully")

	case "down":
		steps := 1
		if len(os.Args) > 2 {
			fmt.Sscanf(os.Args[2], "%d", &steps)
		}
		if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			slog.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations reverted", "steps", steps)

	case "create":
		if len(os.Args) < 3 {
			slog.Error("usage: migrate create <name>")
			os.Exit(1)
		}
		name := os.Args[2]

		upPath := fmt.Sprintf("migrations/%s.up.sql", name)
		downPath := fmt.Sprintf("migrations/%s.down.sql", name)

		if err := os.WriteFile(upPath, []byte(fmt.Sprintf("-- %s: up\n", name)), 0644); err != nil {
			slog.Error("failed to create up file", "error", err)
			os.Exit(1)
		}
		if err := os.WriteFile(downPath, []byte(fmt.Sprintf("-- %s: down\n", name)), 0644); err != nil {
			slog.Error("failed to create down file", "error", err)
			os.Exit(1)
		}

		slog.Info("migration created", "name", name)

	case "status":
		version, dirty, err := m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			slog.Error("failed to get migration version", "error", err)
			os.Exit(1)
		}
		if err == migrate.ErrNilVersion {
			slog.Info("no migrations applied")
		} else {
			slog.Info("migration status", "version", version, "dirty", dirty)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Usage: migrate <command> [args]

Commands:
  up              Apply all pending migrations
  down [steps]    Revert migrations (default: 1)
  create <name>   Create a new migration pair
  status          Show current migration version
`)
}
