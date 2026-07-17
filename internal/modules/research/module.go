package research

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type ResearchModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewResearchModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *ResearchModule {
	return &ResearchModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *ResearchModule) Name() string {
	return m.name
}

func (m *ResearchModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("research module initialized")
	return nil
}

func (m *ResearchModule) Start(ctx context.Context) error {
	return nil
}

func (m *ResearchModule) Stop(ctx context.Context) error {
	return nil
}

func (m *ResearchModule) Service() *Service {
	return m.service
}

func (m *ResearchModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("research module subscribed to event bus")
	}
}

var _ kernel.Module = (*ResearchModule)(nil)
