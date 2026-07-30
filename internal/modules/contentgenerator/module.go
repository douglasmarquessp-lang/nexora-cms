package contentgenerator

import (
	"context"

	"nexora/internal/ai"
	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	publisherModule "nexora/internal/modules/publisher"
	researchModule "nexora/internal/modules/research"
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

func (m *GeneratorModule) SetAIManager(aiManager *ai.Manager) {
	if m.service != nil {
		m.service.SetAIManager(aiManager)
	}
}

func (m *GeneratorModule) SetPublisherSvc(svc *publisherModule.Service) {
	if m.service != nil {
		m.service.SetPublisherSvc(svc)
	}
}

func (m *GeneratorModule) SetResearchSvc(svc *researchModule.Service) {
	if m.service != nil {
		m.service.SetResearchSvc(svc)
	}
}

var _ kernel.Module = (*GeneratorModule)(nil)
