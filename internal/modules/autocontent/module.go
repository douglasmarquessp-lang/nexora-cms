package autocontent

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

type AutocontentModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewAutocontentModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *AutocontentModule {
	return &AutocontentModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *AutocontentModule) Name() string {
	return m.name
}

func (m *AutocontentModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("autocontent module initialized")
	return nil
}

func (m *AutocontentModule) Start(ctx context.Context) error {
	return nil
}

func (m *AutocontentModule) Stop(ctx context.Context) error {
	return nil
}

func (m *AutocontentModule) Service() *Service {
	return m.service
}

func (m *AutocontentModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("autocontent module subscribed to event bus")
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/autocontent", func(r chi.Router) {
		r.Post("/", rest.AdaptHandler(h.CreateJob))
		r.Get("/", rest.AdaptHandler(h.ListJobs))
		r.Get("/stats", rest.AdaptHandler(h.GetStats))
		r.Get("/metrics", rest.AdaptHandler(h.GetMetrics))

		r.Post("/queue", rest.AdaptHandler(h.AddToQueue))
		r.Get("/queue", rest.AdaptHandler(h.ListQueue))
		r.Put("/queue/{queueID}", rest.AdaptHandler(h.UpdateQueueItem))

		r.Post("/templates", rest.AdaptHandler(h.CreateTemplate))
		r.Get("/templates", rest.AdaptHandler(h.ListTemplates))

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", rest.AdaptHandler(h.GetJob))
			r.Put("/", rest.AdaptHandler(h.UpdateJob))
			r.Delete("/", rest.AdaptHandler(h.DeleteJob))
			r.Post("/start", rest.AdaptHandler(h.StartJob))
			r.Post("/pause", rest.AdaptHandler(h.PauseJob))
			r.Post("/resume", rest.AdaptHandler(h.ResumeJob))
			r.Post("/cancel", rest.AdaptHandler(h.CancelJob))
			r.Post("/retry", rest.AdaptHandler(h.RetryStep))
			r.Post("/restart", rest.AdaptHandler(h.RestartJob))
			r.Get("/steps", rest.AdaptHandler(h.GetSteps))
			r.Post("/results", rest.AdaptHandler(h.SaveResult))
			r.Get("/results", rest.AdaptHandler(h.GetResults))
			r.Get("/results/{stepName}", rest.AdaptHandler(h.GetResultByStep))
		})
	})
}

func (m *AutocontentModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("autocontent routes registered")
	return nil
}

var _ kernel.Module = (*AutocontentModule)(nil)
