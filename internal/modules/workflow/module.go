package workflow

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

type WorkflowModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	service  *Service
	eventBus *kernel.EventBus
}

func NewWorkflowModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *WorkflowModule {
	return &WorkflowModule{
		name:  ModuleName,
		cfg:   cfg,
		log:   log,
		db:    db,
		cache: ch,
	}
}

func (m *WorkflowModule) Name() string {
	return m.name
}

func (m *WorkflowModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache)
	m.log.Info("workflow module initialized")
	return nil
}

func (m *WorkflowModule) Start(ctx context.Context) error {
	return nil
}

func (m *WorkflowModule) Stop(ctx context.Context) error {
	return nil
}

func (m *WorkflowModule) Service() *Service {
	return m.service
}

func (m *WorkflowModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("workflow module subscribed to event bus")
	}
}

func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)

	r.Route("/workflow", func(r chi.Router) {
		r.Get("/dashboard", rest.AdaptHandler(h.GetDashboard))
		r.Post("/dashboard/refresh", rest.AdaptHandler(h.RefreshDashboard))

		r.Post("/", rest.AdaptHandler(h.CreateJob))
		r.Get("/", rest.AdaptHandler(h.ListJobs))
		r.Get("/metrics", rest.AdaptHandler(h.GetMetrics))
		r.Get("/stats", rest.AdaptHandler(h.GetStats))
		r.Get("/history", rest.AdaptHandler(h.ListHistory))

		r.Post("/queue", rest.AdaptHandler(h.AddToQueue))
		r.Get("/queue", rest.AdaptHandler(h.ListQueue))
		r.Post("/queue/process", rest.AdaptHandler(h.ProcessQueue))
		r.Put("/queue/{queueID}", rest.AdaptHandler(h.UpdateQueueItem))
		r.Post("/queue/{queueID}/pause", rest.AdaptHandler(h.PauseQueue))
		r.Post("/queue/{queueID}/resume", rest.AdaptHandler(h.ResumeQueue))
		r.Post("/queue/{queueID}/cancel", rest.AdaptHandler(h.CancelQueue))

		r.Get("/notifications", rest.AdaptHandler(h.ListNotifications))
		r.Post("/notifications/read-all", rest.AdaptHandler(h.MarkAllNotificationsRead))
		r.Put("/notifications/{notifID}/read", rest.AdaptHandler(h.MarkNotificationRead))

		r.Post("/actions", rest.AdaptHandler(h.ExecuteAction))

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", rest.AdaptHandler(h.GetJob))
			r.Put("/", rest.AdaptHandler(h.UpdateJob))
			r.Delete("/", rest.AdaptHandler(h.DeleteJob))
			r.Post("/start", rest.AdaptHandler(h.StartJob))
			r.Post("/pause", rest.AdaptHandler(h.PauseJob))
			r.Post("/resume", rest.AdaptHandler(h.ResumeJob))
			r.Post("/cancel", rest.AdaptHandler(h.CancelJob))
			r.Post("/retry", rest.AdaptHandler(h.RetryStep))
			r.Get("/steps", rest.AdaptHandler(h.GetSteps))
			r.Post("/steps/advance", rest.AdaptHandler(h.AdvanceStep))
		})
	})
}

func (m *WorkflowModule) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	m.log.Info("workflow routes registered")
	return nil
}

var _ kernel.Module = (*WorkflowModule)(nil)
