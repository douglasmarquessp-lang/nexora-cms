package tags

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

const ModuleName = "tags"

type TagModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewTagModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *TagModule {
	return &TagModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *TagModule) Name() string {
	return m.name
}

func (m *TagModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("tags module initialized")
	return nil
}

func (m *TagModule) Start(ctx context.Context) error {
	return nil
}

func (m *TagModule) Stop(ctx context.Context) error {
	return nil
}

func (m *TagModule) Service() *Service {
	return m.service
}

func (m *TagModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("tags module subscribed to event bus")
	}
}

var _ kernel.Module = (*TagModule)(nil)
