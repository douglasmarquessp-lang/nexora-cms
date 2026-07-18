package articlepipeline

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

type ArticlePipelineModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewArticlePipelineModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *ArticlePipelineModule {
	return &ArticlePipelineModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *ArticlePipelineModule) Name() string {
	return m.name
}

func (m *ArticlePipelineModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("articlepipeline module initialized")
	return nil
}

func (m *ArticlePipelineModule) Start(ctx context.Context) error {
	return nil
}

func (m *ArticlePipelineModule) Stop(ctx context.Context) error {
	return nil
}

func (m *ArticlePipelineModule) Service() *Service {
	return m.service
}

func (m *ArticlePipelineModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("articlepipeline module subscribed to event bus")
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/articlepipeline", func(r chi.Router) {
		r.Post("/", rest.AdaptHandler(h.CreatePipeline))
		r.Get("/", rest.AdaptHandler(h.ListPipelines))
		r.Get("/candidates", rest.AdaptHandler(h.ListCandidates))
		r.Get("/stats", rest.AdaptHandler(h.GetPipelineStats))

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", rest.AdaptHandler(h.GetPipeline))
			r.Put("/", rest.AdaptHandler(h.UpdatePipeline))
			r.Delete("/", rest.AdaptHandler(h.DeletePipeline))
			r.Post("/start", rest.AdaptHandler(h.StartPipeline))
			r.Post("/pause", rest.AdaptHandler(h.PausePipeline))
			r.Post("/resume", rest.AdaptHandler(h.ResumePipeline))
			r.Post("/cancel", rest.AdaptHandler(h.CancelPipeline))
			r.Post("/retry", rest.AdaptHandler(h.RetryStage))
			r.Post("/restart", rest.AdaptHandler(h.RestartPipeline))

			r.Get("/stages", rest.AdaptHandler(h.GetPipelineStages))
			r.Put("/stages/{stageName}", rest.AdaptHandler(h.UpdateStage))

			r.Post("/metrics", rest.AdaptHandler(h.RecordMetric))
			r.Get("/metrics", rest.AdaptHandler(h.GetPipelineMetrics))

			r.Post("/quality", rest.AdaptHandler(h.CreateQualityReport))
			r.Get("/quality", rest.AdaptHandler(h.GetQualityReports))

			r.Post("/publish", rest.AdaptHandler(h.CreateCandidate))
		})
	})
}

func (m *ArticlePipelineModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("articlepipeline routes registered")
	return nil
}

var _ kernel.Module = (*ArticlePipelineModule)(nil)
