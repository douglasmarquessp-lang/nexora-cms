package editorial

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type EditorialModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewEditorialModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *EditorialModule {
	return &EditorialModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *EditorialModule) Name() string {
	return m.name
}

func (m *EditorialModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("editorial module initialized")
	return nil
}

func (m *EditorialModule) Start(ctx context.Context) error {
	return nil
}

func (m *EditorialModule) Stop(ctx context.Context) error {
	return nil
}

func (m *EditorialModule) Service() *Service {
	return m.service
}

func (m *EditorialModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("editorial module subscribed to event bus")
	}
}

var _ kernel.Module = (*EditorialModule)(nil)
