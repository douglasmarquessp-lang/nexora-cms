package kernel

import (
	"context"
	"fmt"
	"sync"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type Module interface {
	Name() string
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Kernel struct {
	mu       sync.RWMutex
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	modules  map[string]Module
	eventBus *EventBus
	running  bool
}

func New(cfg *config.Config, log *logger.Logger, db *database.Database) *Kernel {
	return &Kernel{
		cfg:      cfg,
		log:      log,
		db:       db,
		modules:  make(map[string]Module),
		eventBus: NewEventBus(log),
		running:  false,
	}
}

func (k *Kernel) RegisterModule(m Module) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	name := m.Name()
	if _, exists := k.modules[name]; exists {
		return fmt.Errorf("module %q already registered", name)
	}

	k.modules[name] = m
	k.log.Info("module registered", "module", name)
	return nil
}

func (k *Kernel) Init(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.log.Info("initializing kernel")

	for _, m := range k.modules {
		if err := m.Init(ctx); err != nil {
			return fmt.Errorf("failed to init module %q: %w", m.Name(), err)
		}
		k.log.Info("module initialized", "module", m.Name())
	}

	return nil
}

func (k *Kernel) Start(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.running {
		return fmt.Errorf("kernel already running")
	}

	k.log.Info("starting kernel")

	for _, m := range k.modules {
		if err := m.Start(ctx); err != nil {
			return fmt.Errorf("failed to start module %q: %w", m.Name(), err)
		}
		k.log.Info("module started", "module", m.Name())
	}

	k.running = true
	k.log.Info("kernel started successfully")
	return nil
}

func (k *Kernel) Stop(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.running {
		return nil
	}

	k.log.Info("stopping kernel")

	for name, m := range k.modules {
		if err := m.Stop(ctx); err != nil {
			k.log.Error("error stopping module", "module", name, "error", err)
		} else {
			k.log.Info("module stopped", "module", name)
		}
	}

	k.running = false
	k.log.Info("kernel stopped")
	return nil
}

func (k *Kernel) Config() *config.Config {
	return k.cfg
}

func (k *Kernel) Logger() *logger.Logger {
	return k.log
}

func (k *Kernel) DB() *database.Database {
	return k.db
}

func (k *Kernel) EventBus() *EventBus {
	return k.eventBus
}

func (k *Kernel) Module(name string) Module {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.modules[name]
}
