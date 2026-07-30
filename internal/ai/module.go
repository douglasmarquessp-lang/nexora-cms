package ai

import (
	"context"
	"fmt"

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
	aiCfg.Enabled = m.cfg.AI.Enabled
	if m.cfg.AI.GlobalTimeout > 0 {
		aiCfg.GlobalTimeout = m.cfg.AI.GlobalTimeout
	}
	if m.cfg.AI.RetryMaxAttempts > 0 {
		aiCfg.Retry.MaxAttempts = m.cfg.AI.RetryMaxAttempts
	}
	if m.cfg.AI.RetryBaseDelay > 0 {
		aiCfg.Retry.BaseDelay = m.cfg.AI.RetryBaseDelay
	}
	if m.cfg.AI.RetryMaxDelay > 0 {
		aiCfg.Retry.MaxDelay = m.cfg.AI.RetryMaxDelay
	}
	if m.cfg.AI.CBFailureThreshold > 0 {
		aiCfg.CircuitBreaker.FailureThreshold = m.cfg.AI.CBFailureThreshold
	}
	if m.cfg.AI.CBRecoveryTimeout > 0 {
		aiCfg.CircuitBreaker.RecoveryTimeout = m.cfg.AI.CBRecoveryTimeout
	}
	if m.cfg.AI.CBHalfOpenMaxReqs > 0 {
		aiCfg.CircuitBreaker.HalfOpenMaxReqs = m.cfg.AI.CBHalfOpenMaxReqs
	}

	m.manager = NewManager(aiCfg, m.log)

	providersRegistered := 0
	for _, p := range m.cfg.AI.Providers {
		provider, err := m.createProvider(p)
		if err != nil {
			m.log.Warn("failed to create AI provider", "name", p.Name, "error", err)
			continue
		}

		internalCfg := ProviderCfg{
			Name:       p.Name,
			Model:      p.Model,
			APIKey:     p.APIKey,
			BaseURL:    p.BaseURL,
			Timeout:    p.Timeout,
			MaxRetries: p.MaxRetries,
			Weight:     p.Weight,
			Priority:   p.Priority,
			Enabled:    p.Enabled,
		}
		if err := m.manager.RegisterProvider(provider, internalCfg); err != nil {
			m.log.Warn("failed to register AI provider", "name", p.Name, "error", err)
			continue
		}
		providersRegistered++
	}

	if providersRegistered == 0 {
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
		} else {
			m.log.Info("registered fallback mock AI provider")
		}
	}

	if m.cfg.AI.DefaultProvider != "" {
		if err := m.manager.SetDefaultProvider(m.cfg.AI.DefaultProvider); err != nil {
			m.log.Warn("failed to set default AI provider", "name", m.cfg.AI.DefaultProvider, "error", err)
		}
	}

	if m.eventBus != nil {
		m.manager.SetEventBus(m.eventBus)
	}

	m.log.Info("AI module initialized", "provider_count", m.manager.Registry().Count())
	return nil
}

func (m *AIModule) createProvider(cfg config.ProviderConfig) (AIProvider, error) {
	switch cfg.Name {
	case "gemini":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("gemini API key is required")
		}
		return NewGeminiProvider(cfg.Name, cfg.Model, cfg.APIKey, cfg.BaseURL), nil
	case "mock":
		return NewMockProvider(cfg.Name, cfg.Model, nil), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Name)
	}
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
