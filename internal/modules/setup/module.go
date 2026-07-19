package setup

import (
	"context"

	"github.com/go-chi/chi/v5"

	"nexora/internal/api/rest"
	"nexora/internal/kernel"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type SetupModule struct {
	name    string
	cfg     *config.Config
	log     *logger.Logger
	db      *database.Database
	service *Service
	eventBus *kernel.EventBus
}

func NewSetupModule(cfg *config.Config, log *logger.Logger, db *database.Database) *SetupModule {
	return &SetupModule{
		name: ModuleName,
		cfg:  cfg,
		log:  log,
		db:   db,
	}
}

func (m *SetupModule) Name() string {
	return m.name
}

func (m *SetupModule) Init(ctx context.Context) error {
	repo := NewRepository(m.db)
	m.service = NewService(m.cfg, m.log, repo)
	m.log.Info("setup module initialized")
	return nil
}

func (m *SetupModule) Start(ctx context.Context) error {
	return nil
}

func (m *SetupModule) Stop(ctx context.Context) error {
	return nil
}

func (m *SetupModule) Service() *Service {
	return m.service
}

func (m *SetupModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/setup", func(r chi.Router) {
		r.Get("/status", rest.AdaptHandler(h.Status))
		r.Post("/install", rest.AdaptHandler(h.Install))
		r.Get("/config", rest.AdaptHandler(h.Config))
		r.Post("/finish", rest.AdaptHandler(h.Finish))
	})
}

func (m *SetupModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("setup routes registered")
	return nil
}

var _ kernel.Module = (*SetupModule)(nil)
