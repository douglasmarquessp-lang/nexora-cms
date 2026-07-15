package site

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

const ModuleName = "site"

type SiteModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewSiteModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *SiteModule {
	return &SiteModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *SiteModule) Name() string {
	return m.name
}

func (m *SiteModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("site module initialized")
	return nil
}

func (m *SiteModule) Start(ctx context.Context) error {
	return nil
}

func (m *SiteModule) Stop(ctx context.Context) error {
	return nil
}

func (m *SiteModule) Service() *Service {
	return m.service
}

func (m *SiteModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("site module subscribed to event bus")
	}
}

var _ kernel.Module = (*SiteModule)(nil)
