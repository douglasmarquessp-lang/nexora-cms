package translation

import (
	"context"

	"github.com/go-chi/chi/v5"

	"nexora/internal/ai"
	"nexora/internal/api/rest"
	"nexora/internal/kernel"
	"nexora/internal/modules/posts"
	"nexora/internal/modules/publisher"
	"nexora/internal/modules/research"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type TranslationModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewTranslationModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *TranslationModule {
	return &TranslationModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *TranslationModule) Name() string {
	return m.name
}

func (m *TranslationModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("translation module initialized")
	return nil
}

func (m *TranslationModule) Start(ctx context.Context) error {
	return nil
}

func (m *TranslationModule) Stop(ctx context.Context) error {
	return nil
}

func (m *TranslationModule) Service() *Service {
	return m.service
}

func (m *TranslationModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("translation module subscribed to event bus")
	}
}

func (m *TranslationModule) SetAIManager(manager *ai.Manager) {
	if m.service != nil {
		m.service.SetAIManager(manager)
	}
}

func (m *TranslationModule) SetQualityChecker(qc ai.QualityChecker) {
	if m.service != nil {
		m.service.SetQualityChecker(qc)
	}
}

func (m *TranslationModule) SetPostsSvc(svc *posts.Service) {
	if m.service != nil {
		m.service.SetPostsSvc(svc)
	}
}

func (m *TranslationModule) SetPublisherSvc(svc *publisher.Service) {
	if m.service != nil {
		m.service.SetPublisherSvc(svc)
	}
}

func (m *TranslationModule) SetResearchSvc(svc *research.Service) {
	if m.service != nil {
		m.service.SetResearchSvc(svc)
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/translation", func(r chi.Router) {
		r.Post("/jobs", rest.AdaptHandler(h.CreateJob))
		r.Get("/jobs", rest.AdaptHandler(h.ListJobs))
		r.Get("/jobs/{id}", rest.AdaptHandler(h.GetJob))
		r.Post("/jobs/{id}/start", rest.AdaptHandler(h.StartJob))
		r.Post("/jobs/{id}/cancel", rest.AdaptHandler(h.CancelJob))
		r.Get("/jobs/{id}/stages", rest.AdaptHandler(h.GetStages))
		r.Get("/jobs/{id}/score", rest.AdaptHandler(h.GetScore))

		r.Post("/stages/{stageID}/approve", rest.AdaptHandler(h.ApproveStage))
		r.Post("/stages/{stageID}/reject", rest.AdaptHandler(h.RejectStage))

		r.Post("/detect", rest.AdaptHandler(h.DetectLanguage))

		r.Post("/glossary", rest.AdaptHandler(h.CreateGlossaryTerm))
		r.Get("/glossary", rest.AdaptHandler(h.ListGlossaryTerms))
		r.Put("/glossary/{termID}", rest.AdaptHandler(h.UpdateGlossaryTerm))
		r.Delete("/glossary/{termID}", rest.AdaptHandler(h.DeleteGlossaryTerm))
	})
}

func (m *TranslationModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("translation routes registered")
	return nil
}

var _ kernel.Module = (*TranslationModule)(nil)
