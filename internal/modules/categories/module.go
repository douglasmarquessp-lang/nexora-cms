package categories

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

const ModuleName = "categories"

type CategoryModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewCategoryModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *CategoryModule {
	return &CategoryModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *CategoryModule) Name() string {
	return m.name
}

func (m *CategoryModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("categories module initialized")
	return nil
}

func (m *CategoryModule) Start(ctx context.Context) error {
	return nil
}

func (m *CategoryModule) Stop(ctx context.Context) error {
	return nil
}

func (m *CategoryModule) Service() *Service {
	return m.service
}

func (m *CategoryModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("categories module subscribed to event bus")
	}
}

var _ kernel.Module = (*CategoryModule)(nil)
