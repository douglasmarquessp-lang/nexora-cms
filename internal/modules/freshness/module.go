package freshness

import (
	"context"

	"github.com/go-chi/chi/v5"

	"nexora/internal/kernel"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type FreshnessModule struct {
	name  string
	cfg   *config.Config
	log   *logger.Logger
	db    *database.Database
	svc   *Service
}

func NewFreshnessModule(cfg *config.Config, log *logger.Logger, db *database.Database) *FreshnessModule {
	return &FreshnessModule{name: ModuleName, cfg: cfg, log: log, db: db}
}

func (m *FreshnessModule) Name() string { return m.name }

func (m *FreshnessModule) Init(ctx context.Context) error {
	m.svc = NewService(m.cfg, m.log, m.db)
	return nil
}

func (m *FreshnessModule) Start(ctx context.Context) error { return nil }
func (m *FreshnessModule) Stop(ctx context.Context) error  { return nil }

func (m *FreshnessModule) Service() *Service { return m.svc }

func (m *FreshnessModule) SetEventBus(bus *kernel.EventBus) {
	if m.svc != nil {
		m.svc.SetEventBus(bus)
	}
}

func (m *FreshnessModule) RegisterRoutes(r chi.Router) error {
	if m.svc == nil {
		return nil
	}
	RegisterRoutes(r, m.svc, m.log)
	return nil
}

var _ kernel.Module = (*FreshnessModule)(nil)
