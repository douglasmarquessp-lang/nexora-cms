package posts

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

const ModuleName = "posts"

type PostModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewPostModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *PostModule {
	return &PostModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *PostModule) Name() string {
	return m.name
}

func (m *PostModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("posts module initialized")
	return nil
}

func (m *PostModule) Start(ctx context.Context) error {
	return nil
}

func (m *PostModule) Stop(ctx context.Context) error {
	return nil
}

func (m *PostModule) Service() *Service {
	return m.service
}

func (m *PostModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("posts module subscribed to event bus")
	}
}

var _ kernel.Module = (*PostModule)(nil)
