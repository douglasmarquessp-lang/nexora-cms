package humanwriter

import (
	"context"

	"github.com/go-chi/chi/v5"

	"nexora/internal/api/rest"
	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type HumanWriterModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewHumanWriterModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *HumanWriterModule {
	return &HumanWriterModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *HumanWriterModule) Name() string {
	return m.name
}

func (m *HumanWriterModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("humanwriter module initialized")
	return nil
}

func (m *HumanWriterModule) Start(ctx context.Context) error {
	return nil
}

func (m *HumanWriterModule) Stop(ctx context.Context) error {
	return nil
}

func (m *HumanWriterModule) Service() *Service {
	return m.service
}

func (m *HumanWriterModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("humanwriter module subscribed to event bus")
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/humanwriter", func(r chi.Router) {
		r.Post("/profiles", rest.AdaptHandler(h.CreateProfile))
		r.Get("/profiles", rest.AdaptHandler(h.ListProfiles))
		r.Get("/profiles/{id}", rest.AdaptHandler(h.GetProfile))
		r.Put("/profiles/{id}", rest.AdaptHandler(h.UpdateProfile))
		r.Delete("/profiles/{id}", rest.AdaptHandler(h.DeleteProfile))

		r.Get("/rules", rest.AdaptHandler(h.ListRules))
		r.Put("/rules/{id}/toggle", rest.AdaptHandler(h.ToggleRule))

		r.Post("/personas", rest.AdaptHandler(h.CreatePersona))
		r.Get("/personas", rest.AdaptHandler(h.ListPersonas))
		r.Get("/personas/{id}", rest.AdaptHandler(h.GetPersona))
		r.Put("/personas/{id}", rest.AdaptHandler(h.UpdatePersona))
		r.Delete("/personas/{id}", rest.AdaptHandler(h.DeletePersona))

		r.Get("/vocabulary", rest.AdaptHandler(h.ListVocabularySets))
		r.Get("/transitions", rest.AdaptHandler(h.ListTransitions))
		r.Get("/patterns", rest.AdaptHandler(h.ListPatterns))
		r.Get("/templates", rest.AdaptHandler(h.ListTemplates))

		r.Post("/humanize", rest.AdaptHandler(h.Humanize))
		r.Post("/batch", rest.AdaptHandler(h.BatchHumanize))
		r.Post("/analyze", rest.AdaptHandler(h.Analyze))

		r.Get("/history", rest.AdaptHandler(h.ListHistory))
		r.Get("/metrics", rest.AdaptHandler(h.GetMetrics))
	})
}

func (m *HumanWriterModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("humanwriter routes registered")
	return nil
}

var _ kernel.Module = (*HumanWriterModule)(nil)
