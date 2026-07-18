package publisher

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

type PublisherModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewPublisherModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *PublisherModule {
	return &PublisherModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *PublisherModule) Name() string {
	return m.name
}

func (m *PublisherModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("publisher module initialized")
	return nil
}

func (m *PublisherModule) Start(ctx context.Context) error {
	return nil
}

func (m *PublisherModule) Stop(ctx context.Context) error {
	return nil
}

func (m *PublisherModule) Service() *Service {
	return m.service
}

func (m *PublisherModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("publisher module subscribed to event bus")
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/publisher", func(r chi.Router) {
		r.Post("/publish", rest.AdaptHandler(h.Publish))
		r.Post("/schedule", rest.AdaptHandler(h.Schedule))
		r.Post("/queue", rest.AdaptHandler(h.AddToQueue))
		r.Get("/queue", rest.AdaptHandler(h.ListQueue))
		r.Get("/schedules", rest.AdaptHandler(h.ListSchedules))
		r.Get("/metrics/summary", rest.AdaptHandler(h.GetMetricsSummary))
		r.Get("/validate-slug", rest.AdaptHandler(h.ValidateSlug))
		r.Get("/generate-slug", rest.AdaptHandler(h.GenerateSlug))

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", rest.AdaptHandler(h.GetPublication))
			r.Put("/", rest.AdaptHandler(h.Update))
			r.Delete("/", rest.AdaptHandler(h.DeletePublication))
			r.Post("/unpublish", rest.AdaptHandler(h.Unpublish))
			r.Post("/republish", rest.AdaptHandler(h.Republish))
			r.Get("/history", rest.AdaptHandler(h.GetHistory))
			r.Get("/metrics", rest.AdaptHandler(h.GetMetrics))

			r.Route("/schedule", func(r chi.Router) {
				r.Get("/", rest.AdaptHandler(h.ListSchedules))
				r.Route("/{scheduleID}", func(r chi.Router) {
					r.Get("/", rest.AdaptHandler(h.GetSchedule))
					r.Post("/cancel", rest.AdaptHandler(h.CancelSchedule))
				})
			})
		})

		r.Route("/queue/{itemID}", func(r chi.Router) {
			r.Post("/retry", rest.AdaptHandler(h.RetryQueue))
		})
	})
}

func (m *PublisherModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("publisher routes registered")
	return nil
}

var _ kernel.Module = (*PublisherModule)(nil)
