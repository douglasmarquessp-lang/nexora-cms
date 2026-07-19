package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	aiModule "nexora/internal/ai"
	articlepipelineModule "nexora/internal/modules/articlepipeline"
	"nexora/internal/api"
	"nexora/internal/api/rest"
	"nexora/internal/kernel"
	assetsModule "nexora/internal/modules/assets"
	authModule "nexora/internal/modules/auth"
	autocontentModule "nexora/internal/modules/autocontent"
	categoriesModule "nexora/internal/modules/categories"
	generatorModule "nexora/internal/modules/contentgenerator"
	editorialModule "nexora/internal/modules/editorial"
	humanwriterModule "nexora/internal/modules/humanwriter"
	editorialEngineModule "nexora/internal/modules/editorialengine"
	mediaModule "nexora/internal/modules/media"
	postsModule "nexora/internal/modules/posts"
	researchModule "nexora/internal/modules/research"
	setupModule "nexora/internal/modules/setup"
	siteModule "nexora/internal/modules/site"
	tagsModule "nexora/internal/modules/tags"
	writerModule "nexora/internal/modules/writer"
	"nexora/internal/pkg/cache"
	casbinPkg "nexora/internal/pkg/casbin"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/ratelimit"
	"nexora/internal/pkg/storage"
	pluginsModule "nexora/internal/plugins"
	publisherModule "nexora/internal/modules/publisher"
	seoengineModule "nexora/internal/modules/seoengine"
	workflowModule "nexora/internal/modules/workflow"
)

type eventBusAdapter struct {
	bus *kernel.EventBus
}

func (a *eventBusAdapter) Emit(ctx context.Context, eventType string, payload interface{}, siteID string) error {
	return a.bus.Emit(ctx, kernel.EventType(eventType), payload, siteID)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg)
	log.Info("starting nexora cms", "version", "0.1.0")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, &cfg.Database, log)
	if err != nil {
		log.Warn("database not available, running in degraded mode", "error", err)
	}

	code := runServer(cfg, log, ctx, db)
	if db != nil {
		db.Close()
	}
	os.Exit(code)
}

func runServer(cfg *config.Config, log *logger.Logger, ctx context.Context, db *database.Database) int {
	ch := cache.New(cfg.Redis.Host == "")

	var enforcer *casbinPkg.Enforcer
	if db != nil && db.Pool != nil {
		if pool, ok := db.Pool.(*pgxpool.Pool); ok {
			e, err := casbinPkg.New(pool, log)
			if err != nil {
				log.Warn("casbin enforcer not available", "error", err)
			} else {
				enforcer = e
			}
		}
	}

	k := kernel.New(cfg, log, db)

	storageDriver := storage.NewDriver(
		cfg.Storage.Driver,
		cfg.Storage.LocalPath,
		"/uploads",
		cfg.Storage.S3Bucket,
		cfg.Storage.S3Region,
		cfg.Storage.S3Endpoint,
		cfg.Storage.S3Key,
		cfg.Storage.S3Secret,
	)

	authMod := authModule.NewAuthModule(cfg, log, db)
	setupMod := setupModule.NewSetupModule(cfg, log, db)
	siteMod := siteModule.NewSiteModule(cfg, log, db, ch)
	postsMod := postsModule.NewPostModule(cfg, log, db, ch)
	categoriesMod := categoriesModule.NewCategoryModule(cfg, log, db, ch)
	tagsMod := tagsModule.NewTagModule(cfg, log, db, ch)
	assetsMod := assetsModule.NewAssetModule(cfg, log, db, ch, storageDriver)
	mediaMod := mediaModule.NewMediaModule(cfg, log, db, ch, storageDriver)
	editorialMod := editorialModule.NewEditorialModule(cfg, log, db, ch)
	researchMod := researchModule.NewResearchModule(cfg, log, db, ch)
	writerMod := writerModule.NewWriterModule(cfg, log, db, ch)
	editorialEngineMod := editorialEngineModule.NewEditorialEngineModule(cfg, log, db, ch)
	generatorMod := generatorModule.NewGeneratorModule(cfg, log, db, ch)
	autocontentMod := autocontentModule.NewAutocontentModule(cfg, log, db, ch)
	humanwriterMod := humanwriterModule.NewHumanWriterModule(cfg, log, db, ch)
	articlepipelineMod := articlepipelineModule.NewArticlePipelineModule(cfg, log, db, ch)
	aiMod := aiModule.NewAIModule(cfg, log, db, ch)
	publisherMod := publisherModule.NewPublisherModule(cfg, log, db, ch)
	seoengineMod := seoengineModule.NewSEOEngineModule(cfg, log, db, ch)
	workflowMod := workflowModule.NewWorkflowModule(cfg, log, db, ch)

	for _, mod := range []kernel.Module{setupMod, authMod, siteMod, postsMod, categoriesMod, tagsMod, assetsMod, mediaMod, editorialMod, researchMod, writerMod, editorialEngineMod, generatorMod, autocontentMod, humanwriterMod, articlepipelineMod, aiMod, publisherMod, seoengineMod, workflowMod} {
		if err := k.RegisterModule(mod); err != nil {
			log.Error("failed to register module", "error", err)
			return 1
		}
	}

	if err := k.Init(ctx); err != nil {
		log.Error("kernel initialization failed", "error", err)
		return 1
	}

	authSvc := authMod.Service()
	authSvc.SetEventBus(k.EventBus())
	authMod.SetEventBus(k.EventBus())

	setupSvc := setupMod.Service()
	setupMod.SetEventBus(k.EventBus())

	siteSvc := siteMod.Service()
	siteSvc.SetEventBus(k.EventBus())

	postsSvc := postsMod.Service()
	postsSvc.SetEventBus(k.EventBus())

	categoriesSvc := categoriesMod.Service()
	categoriesSvc.SetEventBus(k.EventBus())

	tagsSvc := tagsMod.Service()
	tagsSvc.SetEventBus(k.EventBus())

	assetsSvc := assetsMod.Service()
	assetsSvc.SetEventBus(k.EventBus())

	mediaSvc := mediaMod.Service()
	mediaSvc.SetEventBus(k.EventBus())

	editorialSvc := editorialMod.Service()
	editorialSvc.SetEventBus(k.EventBus())

	researchSvc := researchMod.Service()
	researchSvc.SetEventBus(k.EventBus())

	writerSvc := writerMod.Service()
	writerSvc.SetEventBus(k.EventBus())

	editorialEngineSvc := editorialEngineMod.Service()
	editorialEngineSvc.SetEventBus(k.EventBus())

	generatorSvc := generatorMod.Service()
	generatorMod.SetEventBus(k.EventBus())

	autocontentSvc := autocontentMod.Service()
	autocontentMod.SetEventBus(k.EventBus())

	humanwriterSvc := humanwriterMod.Service()
	humanwriterMod.SetEventBus(k.EventBus())

	articlepipelineSvc := articlepipelineMod.Service()
	articlepipelineMod.SetEventBus(k.EventBus())

	aiSvc := aiMod.Service()
	aiMod.SetEventBus(k.EventBus())

	publisherSvc := publisherMod.Service()
	publisherMod.SetEventBus(k.EventBus())

	seoengineSvc := seoengineMod.Service()
	seoengineMod.SetEventBus(k.EventBus())

	workflowSvc := workflowMod.Service()
	workflowMod.SetEventBus(k.EventBus())

	pluginManager := pluginsModule.NewManager(&pluginsModule.ManagerConfig{
		PluginsDir: "plugins",
	}, log, &eventBusAdapter{bus: k.EventBus()})
	if err := pluginManager.Init(ctx); err != nil {
		log.Warn("plugin manager initialization", "error", err)
	}

	if err := k.Start(ctx); err != nil {
		log.Error("kernel start failed", "error", err)
		return 1
	}

	rateLimitStore := ratelimit.NewMemoryStore()
	rateLimiter := ratelimit.NewLimiter(rateLimitStore, ratelimit.Config{
		Enabled:     true,
		MaxRequests: 100,
		Window:      time.Minute,
	})

	router := rest.NewRouter(log)

	dbPing := func(ctx context.Context) error {
		if db != nil && db.Pool != nil {
			return db.Pool.Ping(ctx)
		}
		return fmt.Errorf("database not connected")
	}

	var dbExec interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	}
	if db != nil {
		dbExec = db.Pool
	}

	deps := &api.Dependencies{
		Log:                log,
		DBPing:             dbPing,
		DBExec:             dbExec,
		AuthSvc:            authSvc,
		SetupSvc:           setupSvc,
		SiteSvc:            siteSvc,
		PostsSvc:           postsSvc,
		CategoriesSvc:      categoriesSvc,
		TagsSvc:            tagsSvc,
		AssetsSvc:          assetsSvc,
		MediaSvc:           mediaSvc,
		EditorialSvc:       editorialSvc,
		ResearchSvc:        researchSvc,
		WriterSvc:          writerSvc,
		EditorialEngineSvc: editorialEngineSvc,
		GeneratorSvc:       generatorSvc,
		AutocontentSvc:     autocontentSvc,
		HumanWriterSvc:     humanwriterSvc,
		ArticlePipelineSvc: articlepipelineSvc,
		AIManager:          aiSvc,
		PublisherSvc:       publisherSvc,
		SeoEngineSvc:       seoengineSvc,
		WorkflowSvc:        workflowSvc,
		PluginManager:      pluginManager,
		CasbinEnforcer:     enforcer,
		RateLimits:         rateLimiter,
	}

	api.SetupRoutes(router, deps)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    cfg.Server.Timeout,
		WriteTimeout:   cfg.Server.Timeout,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Info("api server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced shutdown", "error", err)
	}

	if err := k.Stop(shutdownCtx); err != nil {
		log.Error("kernel stop error", "error", err)
	}

	log.Info("server stopped")
	return 0
}
