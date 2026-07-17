package writer

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type WriterModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewWriterModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *WriterModule {
	return &WriterModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *WriterModule) Name() string {
	return m.name
}

func (m *WriterModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("writer module initialized")
	return nil
}

func (m *WriterModule) Start(ctx context.Context) error {
	return nil
}

func (m *WriterModule) Stop(ctx context.Context) error {
	return nil
}

func (m *WriterModule) Service() *Service {
	return m.service
}

func (m *WriterModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("writer module subscribed to event bus")
	}
}

var _ kernel.Module = (*WriterModule)(nil)
