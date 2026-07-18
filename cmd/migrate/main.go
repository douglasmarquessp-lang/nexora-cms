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
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env file not found, using environment variables", "error", err)
	}

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

	code := runCommand(m, command)
	m.Close()
	os.Exit(code)
}

func runCommand(m *migrate.Migrate, command string) int {
	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			slog.Error("migration up failed", "error", err)
			return 1
		}
		slog.Info("migrations applied successfully")

	case "down":
		steps := 1
		if len(os.Args) > 2 {
			if n, err := fmt.Sscanf(os.Args[2], "%d", &steps); err != nil || n != 1 {
				slog.Warn("invalid step count, using default", "value", os.Args[2])
				steps = 1
			}
		}
		if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			slog.Error("migration down failed", "error", err)
			return 1
		}
		slog.Info("migrations reverted", "steps", steps)

	case "create":
		if len(os.Args) < 3 {
			slog.Error("usage: migrate create <name>")
			return 1
		}
		name := os.Args[2]

		upPath := fmt.Sprintf("migrations/%s.up.sql", name)
		downPath := fmt.Sprintf("migrations/%s.down.sql", name)

		if err := os.WriteFile(upPath, []byte(fmt.Sprintf("-- %s: up\n", name)), 0o644); err != nil {
			slog.Error("failed to create up file", "error", err)
			return 1
		}
		if err := os.WriteFile(downPath, []byte(fmt.Sprintf("-- %s: down\n", name)), 0o644); err != nil {
			slog.Error("failed to create down file", "error", err)
			return 1
		}

		slog.Info("migration created", "name", name)

	case "status":
		version, dirty, err := m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			slog.Error("failed to get migration version", "error", err)
			return 1
		}
		if err == migrate.ErrNilVersion {
			slog.Info("no migrations applied")
		} else {
			slog.Info("migration status", "version", version, "dirty", dirty)
		}

	default:
		printUsage()
		return 1
	}
	return 0
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
