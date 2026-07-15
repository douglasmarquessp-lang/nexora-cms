package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

type Pool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

type Database struct {
	Pool Pool
}

func New(ctx context.Context, cfg *config.DatabaseConfig, log *logger.Logger) (*Database, error) {
	log.Info("connecting to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Name,
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
	)

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = int32(cfg.MinConns)

	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection established",
		"max_conns", poolCfg.MaxConns,
		"min_conns", poolCfg.MinConns,
	)

	return &Database{Pool: pool}, nil
}

func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
