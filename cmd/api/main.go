package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nexora/internal/api"
	"nexora/internal/api/rest"
	"nexora/internal/kernel"
	authModule "nexora/internal/modules/auth"
	categoriesModule "nexora/internal/modules/categories"
	postsModule "nexora/internal/modules/posts"
	siteModule "nexora/internal/modules/site"
	tagsModule "nexora/internal/modules/tags"
	"nexora/internal/pkg/cache"
	casbinPkg "nexora/internal/pkg/casbin"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/ratelimit"
)

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

	if db != nil {
		defer db.Close()
	}

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

	authMod := authModule.NewAuthModule(cfg, log, db)
	siteMod := siteModule.NewSiteModule(cfg, log, db, ch)
	postsMod := postsModule.NewPostModule(cfg, log, db, ch)
	categoriesMod := categoriesModule.NewCategoryModule(cfg, log, db, ch)
	tagsMod := tagsModule.NewTagModule(cfg, log, db, ch)

	for _, mod := range []kernel.Module{authMod, siteMod, postsMod, categoriesMod, tagsMod} {
		if err := k.RegisterModule(mod); err != nil {
			log.Error("failed to register module", "error", err)
			os.Exit(1)
		}
	}

	if err := k.Init(ctx); err != nil {
		log.Error("kernel initialization failed", "error", err)
		os.Exit(1)
	}

	authSvc := authMod.Service()
	authSvc.SetEventBus(k.EventBus())
	authMod.SetEventBus(k.EventBus())

	siteSvc := siteMod.Service()
	siteSvc.SetEventBus(k.EventBus())

	postsSvc := postsMod.Service()
	postsSvc.SetEventBus(k.EventBus())

	categoriesSvc := categoriesMod.Service()
	categoriesSvc.SetEventBus(k.EventBus())

	tagsSvc := tagsMod.Service()
	tagsSvc.SetEventBus(k.EventBus())

	if err := k.Start(ctx); err != nil {
		log.Error("kernel start failed", "error", err)
		os.Exit(1)
	}

	rateLimitStore := ratelimit.NewMemoryStore()
	rateLimiter := ratelimit.NewLimiter(rateLimitStore, ratelimit.Config{
		Enabled:      true,
		MaxRequests: 100,
		Window:       time.Minute,
	})

	router := rest.NewRouter(log)

	dbPing := func(ctx context.Context) error {
		if db != nil && db.Pool != nil {
			return db.Pool.Ping(ctx)
		}
		return fmt.Errorf("database not connected")
	}

	deps := &api.Dependencies{
		Log:            log,
		DBPing:         dbPing,
		DBExec:         db.Pool,
		AuthSvc:        authSvc,
		SiteSvc:        siteSvc,
		PostsSvc:       postsSvc,
		CategoriesSvc:  categoriesSvc,
		TagsSvc:        tagsSvc,
		CasbinEnforcer: enforcer,
		RateLimits:     rateLimiter,
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
			os.Exit(1)
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
}
