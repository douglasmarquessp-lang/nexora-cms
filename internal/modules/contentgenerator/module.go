package contentgenerator

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type GeneratorModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewGeneratorModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *GeneratorModule {
	return &GeneratorModule{
		name:  "contentgenerator",
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *GeneratorModule) Name() string {
	return m.name
}

func (m *GeneratorModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("content generator module initialized")
	return nil
}

func (m *GeneratorModule) Start(ctx context.Context) error {
	return nil
}

func (m *GeneratorModule) Stop(ctx context.Context) error {
	return nil
}

func (m *GeneratorModule) Service() *Service {
	return m.service
}

func (m *GeneratorModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("content generator module subscribed to event bus")
	}
}

var _ kernel.Module = (*GeneratorModule)(nil)
