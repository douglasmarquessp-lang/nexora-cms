package assets

import (
	"context"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/storage"
)

const ModuleName = "assets"

type AssetModule struct {
	name     string
	cfg      *config.Config
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	storage  storage.Driver
	service  *Service
	eventBus *kernel.EventBus
}

func NewAssetModule(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache, st storage.Driver) *AssetModule {
	return &AssetModule{
		name:    ModuleName,
		cfg:     cfg,
		log:     log,
		db:      db,
		cache:   ch,
		storage: st,
	}
}

func (m *AssetModule) Name() string {
	return m.name
}

func (m *AssetModule) Init(ctx context.Context) error {
	m.service = NewService(m.cfg, m.log, m.db, m.cache, m.storage)
	m.log.Info("assets module initialized")
	return nil
}

func (m *AssetModule) Start(ctx context.Context) error {
	return nil
}

func (m *AssetModule) Stop(ctx context.Context) error {
	return nil
}

func (m *AssetModule) Service() *Service {
	return m.service
}

func (m *AssetModule) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
	if m.service != nil {
		m.service.SetEventBus(bus)
	}
	if bus != nil {
		m.log.Info("assets module subscribed to event bus")
	}
}

var _ kernel.Module = (*AssetModule)(nil)
