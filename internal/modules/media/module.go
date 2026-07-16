package media

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/storage"
)

const ModuleName = "media"

type MediaModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	storage  storage.Driver
	service  *Service
	eventBus *kernel.EventBus
}

func NewMediaModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache, st storage.Driver) *MediaModule {
	return &MediaModule{
		name:    ModuleName,
		cfg:     cfg,
		log:     log,
		db:      db,
		cache:   ch,
		storage: st,
	}
}

func (m *MediaModule) Name() string {
	return m.name
}

func (m *MediaModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache, m.storage)
	m.log.Info("media module initialized")
	return nil
}

func (m *MediaModule) Start(ctx context.Context) error {
	return nil
}

func (m *MediaModule) Stop(ctx context.Context) error {
	return nil
}

func (m *MediaModule) Service() *Service {
	return m.service
}

func (m *MediaModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("media module subscribed to event bus")
	}
}

var _ kernel.Module = (*MediaModule)(nil)
