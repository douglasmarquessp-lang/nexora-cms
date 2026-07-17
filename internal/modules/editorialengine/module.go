package editorialengine

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type EditorialEngineModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewEditorialEngineModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *EditorialEngineModule {
	return &EditorialEngineModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *EditorialEngineModule) Name() string {
	return m.name
}

func (m *EditorialEngineModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("editorial engine module initialized")
	return nil
}

func (m *EditorialEngineModule) Start(ctx context.Context) error {
	return nil
}

func (m *EditorialEngineModule) Stop(ctx context.Context) error {
	return nil
}

func (m *EditorialEngineModule) Service() *Service {
	return m.service
}

func (m *EditorialEngineModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("editorial engine module subscribed to event bus")
	}
}

var _ kernel.Module = (*EditorialEngineModule)(nil)
