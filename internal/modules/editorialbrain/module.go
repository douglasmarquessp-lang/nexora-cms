// Package editorialbrain implements the AI Editorial Brain: before any
// article is written it builds an intelligent editorial brief (search intent,
// persona, outline, required questions) and before publication it produces a
// full editorial note (coverage, fluency, evidence, per-block confidence,
// semantic SEO) with a weighted final score and an approve/review decision.
package editorialbrain

import (
	"context"

	"github.com/go-chi/chi/v5"

	"nexora/internal/ai"
	"nexora/internal/kernel"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

// Module is the kernel module for the editorial brain.
type Module struct {
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	service  *Service
	eventBus *kernel.EventBus
}

// NewEditorialBrainModule creates the kernel module. DB may be nil
// (deterministic engines still work fully).
func NewEditorialBrainModule(cfg *config.Config, log *logger.Logger, db *database.Database) *Module {
	return &Module{
		cfg:     cfg,
		log:     log,
		db:      db,
		service: NewService(cfg, log, db),
	}
}

// Name implements kernel.Module.
func (m *Module) Name() string { return "editorialbrain" }

// Service returns the module service.
func (m *Module) Service() *Service { return m.service }

// SetEventBus wires the module event bus.
func (m *Module) SetEventBus(bus *kernel.EventBus) {
	m.service.SetEventBus(bus)
	m.eventBus = bus
}

// SetQualityChecker wires the deterministic AI quality checker.
func (m *Module) SetQualityChecker(qc ai.QualityChecker) {
	m.service.SetQualityChecker(qc)
}

// SetResearchProvider wires the research fact/source provider.
func (m *Module) SetResearchProvider(rp ResearchProvider) {
	m.service.SetResearchProvider(rp)
}

// RegisterRoutes registers the module REST routes.
func (m *Module) RegisterRoutes(r chi.Router) error {
	RegisterRoutes(r, m.service, m.log)
	return nil
}

// Init implements kernel.Module.
func (m *Module) Init(ctx context.Context) error { return nil }

// Start implements kernel.Module.
func (m *Module) Start(ctx context.Context) error { return nil }

// Stop implements kernel.Module.
func (m *Module) Stop(ctx context.Context) error { return nil }

var _ kernel.Module = (*Module)(nil)
