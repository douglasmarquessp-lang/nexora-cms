package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"nexora/internal/api/health"
	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	assetsModule "nexora/internal/modules/assets"
	authModule "nexora/internal/modules/auth"
	categoriesModule "nexora/internal/modules/categories"
	mediaModule "nexora/internal/modules/media"
	pluginsModule "nexora/internal/plugins"
	postsModule "nexora/internal/modules/posts"
	siteModule "nexora/internal/modules/site"
	tagsModule "nexora/internal/modules/tags"
	casbinPkg "nexora/internal/pkg/casbin"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/ratelimit"
)

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type pingFunc func(ctx context.Context) error

type Dependencies struct {
	Log              *logger.Logger
	DBPing           pingFunc
	DBExec           dbExecutor
	AuthSvc          *authModule.Service
	SiteSvc          *siteModule.Service
	PostsSvc         *postsModule.Service
	CategoriesSvc    *categoriesModule.Service
	TagsSvc          *tagsModule.Service
	AssetsSvc        *assetsModule.Service
	MediaSvc         *mediaModule.Service
	PluginManager    *pluginsModule.Manager
	CasbinEnforcer   *casbinPkg.Enforcer
	RateLimits       *ratelimit.Limiter
}

func SetupRoutes(router *rest.Router, deps *Dependencies) {
	healthHandler := health.NewHandler(health.CheckFunc(deps.DBPing))

	authHandler := authModule.NewHandler(deps.AuthSvc, deps.Log)
	authMiddleware := middleware.RequireAuth(deps.AuthSvc)

	siteHandler := siteModule.NewHandler(deps.SiteSvc, deps.Log)
	siteIdentify := middleware.IdentifySite(deps.SiteSvc)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", rest.AdaptHandler(healthHandler.Check))

		r.Group(func(r chi.Router) {
			if deps.RateLimits != nil {
				r.Use(deps.RateLimits.Middleware)
			}

			r.Post("/auth/register", rest.AdaptHandler(authHandler.Register))
			r.Post("/auth/login", rest.AdaptHandler(authHandler.Login))
			r.Post("/auth/refresh", rest.AdaptHandler(authHandler.RefreshToken))
			r.Post("/auth/logout", wrapMiddleware(authMiddleware, rest.AdaptHandler(authHandler.Logout)))
			r.Get("/auth/me", wrapMiddleware(authMiddleware, rest.AdaptHandler(authHandler.Me)))
			r.Get("/auth/oauth/url", rest.AdaptHandler(authHandler.GetOAuthURL))
			r.Post("/auth/oauth/callback", rest.AdaptHandler(authHandler.OAuthCallback))
			r.Post("/auth/mfa/enroll", wrapMiddleware(authMiddleware, rest.AdaptHandler(authHandler.EnrollMFA)))
			r.Post("/auth/mfa/verify", wrapMiddleware(authMiddleware, rest.AdaptHandler(authHandler.VerifyMFA)))
			r.Post("/auth/mfa/disable", wrapMiddleware(authMiddleware, rest.AdaptHandler(authHandler.DisableMFA)))
		})

		r.Group(func(r chi.Router) {
			r.Use(siteIdentify)
			r.Use(authMiddleware)
			r.Use(middleware.RLSContext(deps.SiteSvc, deps.DBExec))

			r.Get("/sites", rest.AdaptHandler(siteHandler.List))
			r.Post("/sites", rest.AdaptHandler(siteHandler.Create))
			r.Get("/sites/{id}", rest.AdaptHandler(siteHandler.Get))
			r.Put("/sites/{id}", rest.AdaptHandler(siteHandler.Update))
			r.Delete("/sites/{id}", rest.AdaptHandler(siteHandler.Delete))

			r.Get("/sites/{id}/domains", rest.AdaptHandler(siteHandler.ListDomains))
			r.Post("/sites/{id}/domains", rest.AdaptHandler(siteHandler.AddDomain))
			r.Delete("/sites/{id}/domains/{domainID}", rest.AdaptHandler(siteHandler.RemoveDomain))
			r.Put("/sites/{id}/domains/{domainID}/primary", rest.AdaptHandler(siteHandler.SetPrimaryDomain))

			r.Get("/sites/{id}/settings", rest.AdaptHandler(siteHandler.ListSiteSettings))
			r.Get("/sites/{id}/settings/{key}", rest.AdaptHandler(siteHandler.GetSiteSetting))
			r.Put("/sites/{id}/settings", rest.AdaptHandler(siteHandler.SetSiteSetting))

			r.Get("/system/config", rest.AdaptHandler(siteHandler.ListGlobalSettings))
			r.Get("/system/config/{key}", rest.AdaptHandler(siteHandler.GetGlobalSetting))
			r.Put("/system/config", rest.AdaptHandler(siteHandler.SetGlobalSetting))

			registerContentRoutes(r, deps)
		})
	})
}

func registerContentRoutes(r chi.Router, deps *Dependencies) {
	postsHandler := postsModule.NewHandler(deps.PostsSvc, deps.Log)
	categoriesHandler := categoriesModule.NewHandler(deps.CategoriesSvc, deps.Log)
	tagsHandler := tagsModule.NewHandler(deps.TagsSvc, deps.Log)
	assetsHandler := assetsModule.NewHandler(deps.AssetsSvc, deps.Log)
	mediaHandler := mediaModule.NewHandler(deps.MediaSvc, deps.Log)
	pluginHandler := pluginsModule.NewHandler(deps.PluginManager)

	r.Get("/posts", rest.AdaptHandler(postsHandler.List))
	r.Post("/posts", rest.AdaptHandler(postsHandler.Create))
	r.Get("/posts/{id}", rest.AdaptHandler(postsHandler.Get))
	r.Put("/posts/{id}", rest.AdaptHandler(postsHandler.Update))
	r.Delete("/posts/{id}", rest.AdaptHandler(postsHandler.Delete))
	r.Patch("/posts/{id}/status", rest.AdaptHandler(postsHandler.SetStatus))
	r.Post("/posts/{id}/autosave", rest.AdaptHandler(postsHandler.Autosave))
	r.Get("/posts/{id}/autosave", rest.AdaptHandler(postsHandler.GetAutosave))
	r.Delete("/posts/{id}/autosave", rest.AdaptHandler(postsHandler.DeleteAutosave))

	r.Get("/categories", rest.AdaptHandler(categoriesHandler.List))
	r.Get("/categories/tree", rest.AdaptHandler(categoriesHandler.Tree))
	r.Post("/categories", rest.AdaptHandler(categoriesHandler.Create))
	r.Get("/categories/{id}", rest.AdaptHandler(categoriesHandler.Get))
	r.Put("/categories/{id}", rest.AdaptHandler(categoriesHandler.Update))
	r.Delete("/categories/{id}", rest.AdaptHandler(categoriesHandler.Delete))

	r.Get("/tags", rest.AdaptHandler(tagsHandler.List))
	r.Post("/tags", rest.AdaptHandler(tagsHandler.Create))
	r.Get("/tags/{id}", rest.AdaptHandler(tagsHandler.Get))
	r.Put("/tags/{id}", rest.AdaptHandler(tagsHandler.Update))
	r.Delete("/tags/{id}", rest.AdaptHandler(tagsHandler.Delete))

	r.Get("/assets", rest.AdaptHandler(assetsHandler.List))
	r.Post("/assets/upload", rest.AdaptHandler(assetsHandler.Upload))
	r.Get("/assets/{id}", rest.AdaptHandler(assetsHandler.Get))
	r.Put("/assets/{id}", rest.AdaptHandler(assetsHandler.Update))
	r.Delete("/assets/{id}", rest.AdaptHandler(assetsHandler.Delete))
	r.Post("/assets/link", rest.AdaptHandler(assetsHandler.LinkToPost))
	r.Delete("/assets/{postID}/link/{assetID}", rest.AdaptHandler(assetsHandler.UnlinkFromPost))
	r.Get("/posts/{postID}/assets", rest.AdaptHandler(assetsHandler.GetPostAssets))

	mediaModule.RegisterRoutes(r, mediaHandler, deps.CasbinEnforcer)
	pluginsModule.RegisterRoutes(r, pluginHandler)
}

func wrapMiddleware(mw func(http.Handler) http.Handler, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mw(handler).ServeHTTP(w, r)
	}
}
