// Package migrate applies database migrations at application startup.
package migrate

import (
	"context"
	"errors"
	"fmt"

	golangmigrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"

	"nexora/internal/pkg/logger"
)

// advisoryLockID is a fixed identifier used to serialize migration runs
// across concurrent API instances. It must stay distinct from the advisory
// lock ID that golang-migrate derives internally (GenerateAdvisoryLockId),
// otherwise a process would block on its own lock and time out.
const advisoryLockID = int64(7645017289165482947)

// Run applies all pending migrations located in migrationsDir against the
// database referenced by dsn. It acquires a PostgreSQL advisory lock first so
// that concurrent instances (or the manual migrate CLI) never apply the same
// migration twice. The lock is released after migrations finish, including on
// failure. Any migration error — including a dirty schema_migrations state —
// is returned so the caller can abort startup.
func Run(ctx context.Context, dsn, migrationsDir string, log *logger.Logger) error {
	lockConn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect for migration lock: %w", err)
	}
	defer func() { _ = lockConn.Close(context.Background()) }()

	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	defer func() { _, _ = lockConn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID) }()

	log.Info("migration advisory lock acquired")

	m, err := golangmigrate.New("file://"+migrationsDir, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, golangmigrate.ErrNoChange) {
		return fmt.Errorf("migration failed: %w", err)
	}

	if version, dirty, vErr := m.Version(); vErr != nil && !errors.Is(vErr, golangmigrate.ErrNilVersion) {
		return fmt.Errorf("failed to read migration version: %w", vErr)
	} else if vErr == nil {
		log.Info("migrations up to date", "version", version, "dirty", dirty)
	}

	return nil
}
