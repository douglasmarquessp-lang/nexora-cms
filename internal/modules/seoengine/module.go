package seoengine

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

type SEOEngineModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewSEOEngineModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *SEOEngineModule {
	return &SEOEngineModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *SEOEngineModule) Name() string {
	return m.name
}

func (m *SEOEngineModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("seoengine module initialized")
	return nil
}

func (m *SEOEngineModule) Start(ctx context.Context) error {
	return nil
}

func (m *SEOEngineModule) Stop(ctx context.Context) error {
	return nil
}

func (m *SEOEngineModule) Service() *Service {
	return m.service
}

func (m *SEOEngineModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("seoengine module subscribed to event bus")
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/seoengine", func(r chi.Router) {
		r.Post("/", rest.AdaptHandler(h.CreateProject))
		r.Get("/", rest.AdaptHandler(h.ListProjects))
		r.Get("/stats", rest.AdaptHandler(h.GetDashboardStats))
		r.Get("/metrics", rest.AdaptHandler(h.GetMetrics))
		r.Get("/orphans", rest.AdaptHandler(h.DetectOrphans))
		r.Get("/cannibalization", rest.AdaptHandler(h.DetectCannibalization))
		r.Get("/content-gaps", rest.AdaptHandler(h.DetectContentGaps))

		r.Post("/clusters", rest.AdaptHandler(h.CreateCluster))
		r.Get("/clusters", rest.AdaptHandler(h.ListClusters))

		r.Post("/keywords/analyze", rest.AdaptHandler(h.AnalyzeKeywords))

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", rest.AdaptHandler(h.GetProject))
			r.Put("/", rest.AdaptHandler(h.UpdateProject))
			r.Delete("/", rest.AdaptHandler(h.DeleteProject))

			r.Post("/audit", rest.AdaptHandler(h.RunFullAudit))
			r.Get("/audits", rest.AdaptHandler(h.GetProjectAudits))
			r.Get("/audits/{auditID}", rest.AdaptHandler(h.GetAudit))

			r.Post("/content-analysis", rest.AdaptHandler(h.AnalyzeContent))
			r.Post("/technical-analysis", rest.AdaptHandler(h.AnalyzeTechnical))

			r.Get("/scores", rest.AdaptHandler(h.GetScores))

			r.Post("/improvements", rest.AdaptHandler(h.AddImprovement))
			r.Get("/improvements", rest.AdaptHandler(h.ListImprovements))
			r.Put("/improvements/{improvementID}", rest.AdaptHandler(h.UpdateImprovement))

			r.Get("/linking-suggestions", rest.AdaptHandler(h.GetLinkingSuggestions))
			r.Get("/schema-recommendations", rest.AdaptHandler(h.GetSchemaRecommendations))
			r.Post("/checklist", rest.AdaptHandler(h.GenerateChecklist))

			r.Get("/duplicates", rest.AdaptHandler(h.DetectDuplicates))
		})
	})
}

func (m *SEOEngineModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("seoengine routes registered")
	return nil
}

var _ kernel.Module = (*SEOEngineModule)(nil)
