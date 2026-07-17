package ai

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

const ModuleName = "ai"

type AIModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	manager  *Manager
	eventBus *kernel.EventBus
}

func NewAIModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *AIModule {
	return &AIModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *AIModule) Name() string {
	return m.name
}

func (m *AIModule) Init(ctx context.Context) error {
	aiCfg := DefaultConfig()
	m.manager = NewManager(aiCfg, m.log)

	providerCfg := ProviderCfg{
		Name:       "mock",
		Model:      "mock-model",
		Priority:   1,
		Weight:     10,
		Enabled:    true,
		MaxRetries: 3,
	}
	provider := NewMockProvider("mock", "mock-model", nil)
	if err := m.manager.RegisterProvider(provider, providerCfg); err != nil {
		m.log.Warn("failed to register mock AI provider", "error", err)
	}

	if m.eventBus != nil {
		m.manager.SetEventBus(m.eventBus)
	}

	m.log.Info("AI module initialized", "provider_count", m.manager.Registry().Count())
	return nil
}

func (m *AIModule) Start(ctx context.Context) error {
	return nil
}

func (m *AIModule) Stop(ctx context.Context) error {
	return nil
}

func (m *AIModule) Service() *Manager {
	return m.manager
}

func (m *AIModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.manager != nil {
		m.manager.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("AI module subscribed to event bus")
	}
}

var _ kernel.Module = (*AIModule)(nil)
