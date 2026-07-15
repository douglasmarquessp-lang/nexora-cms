package auth

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

const ModuleName = "auth"

type AuthModule struct {
	name    string
	cfg     *config.Config
	log     *logger.Logger
	db      *database.Database
	service *Service
	eventBus *kernel.EventBus
}

func NewAuthModule(cfg *config.Config, log *logger.Logger, db *database.Database) *AuthModule {
	return &AuthModule{
		name: ModuleName,
		cfg:  cfg,
		log:  log,
		db:   db,
	}
}

func (m *AuthModule) Name() string {
	return m.name
}

func (m *AuthModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db)
	m.log.Info("auth module initialized")
	return nil
}

func (m *AuthModule) Start(ctx context.Context) error {
	return nil
}

func (m *AuthModule) Stop(ctx context.Context) error {
	return nil
}

func (m *AuthModule) Service() *Service {
	return m.service
}

func (m *AuthModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if bus != nil {
		bus.Subscribe(kernel.EventUserRegistered, m.handleEvent)
		bus.Subscribe(kernel.EventUserLogin, m.handleEvent)
		m.log.Info("auth module subscribed to events")
	}
}

func (m *AuthModule) handleEvent(ctx context.Context, event kernel.Event) error {
	m.log.Debug("auth module received event",
		"event_type", string(event.Type),
		"event_id", event.ID,
	)
	return nil
}

var _ kernel.Module = (*AuthModule)(nil)
