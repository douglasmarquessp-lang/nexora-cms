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
	editorialModule "nexora/internal/modules/editorial"
	postsModule "nexora/internal/modules/posts"
	researchModule "nexora/internal/modules/research"
	writerModule "nexora/internal/modules/writer"
	editorialEngineModule "nexora/internal/modules/editorialengine"
	generatorModule "nexora/internal/modules/contentgenerator"
	autocontentModule "nexora/internal/modules/autocontent"
	aiModule "nexora/internal/ai"
	articlepipelineModule "nexora/internal/modules/articlepipeline"
	humanwriterModule "nexora/internal/modules/humanwriter"
	publisherModule "nexora/internal/modules/publisher"
	seoengineModule "nexora/internal/modules/seoengine"
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
	EditorialSvc     *editorialModule.Service
	ResearchSvc      *researchModule.Service
	WriterSvc        *writerModule.Service
	EditorialEngineSvc *editorialEngineModule.Service
	GeneratorSvc          *generatorModule.Service
	AutocontentSvc        *autocontentModule.Service
	HumanWriterSvc        *humanwriterModule.Service
	ArticlePipelineSvc    *articlepipelineModule.Service
	PublisherSvc          *publisherModule.Service
	SeoEngineSvc          *seoengineModule.Service
	AIManager             *aiModule.Manager
	PluginManager      *pluginsModule.Manager
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

	registerEditorialRoutes(r, deps)
	registerEditorialEngineRoutes(r, deps)
	registerGeneratorRoutes(r, deps)
	registerAutocontentRoutes(r, deps)
	registerHumanWriterRoutes(r, deps)
	registerArticlePipelineRoutes(r, deps)
	registerPublisherRoutes(r, deps)
	registerSeoEngineRoutes(r, deps)
	registerAIRoutes(r, deps)
}

func registerEditorialRoutes(r chi.Router, deps *Dependencies) {
	editorialHandler := editorialModule.NewHandler(deps.EditorialSvc, deps.Log)

	r.Get("/editorial/dashboard", rest.AdaptHandler(editorialHandler.Dashboard))
	r.Get("/editorial/stats", rest.AdaptHandler(editorialHandler.Stats))
	r.Get("/editorial/posts/recent", rest.AdaptHandler(editorialHandler.RecentPosts))
	r.Get("/editorial/posts/drafts", rest.AdaptHandler(editorialHandler.DraftPosts))
	r.Get("/editorial/posts/scheduled", rest.AdaptHandler(editorialHandler.ScheduledPosts))

	r.Get("/editorial/tasks", rest.AdaptHandler(editorialHandler.ListTasks))
	r.Post("/editorial/tasks", rest.AdaptHandler(editorialHandler.CreateTask))
	r.Get("/editorial/tasks/{id}", rest.AdaptHandler(editorialHandler.GetTask))
	r.Put("/editorial/tasks/{id}", rest.AdaptHandler(editorialHandler.UpdateTask))
	r.Delete("/editorial/tasks/{id}", rest.AdaptHandler(editorialHandler.DeleteTask))
	r.Get("/editorial/tasks/date/{date}", rest.AdaptHandler(editorialHandler.GetTasksForDate))

	r.Post("/editorial/posts/{postID}/revisions", rest.AdaptHandler(editorialHandler.SaveRevision))
	r.Get("/editorial/posts/{postID}/revisions", rest.AdaptHandler(editorialHandler.ListRevisions))
	r.Post("/editorial/posts/{postID}/revisions/{revID}/restore", rest.AdaptHandler(editorialHandler.RestoreRevision))

	r.Post("/editorial/posts/{postID}/approvals", rest.AdaptHandler(editorialHandler.RequestApproval))
	r.Put("/editorial/posts/{postID}/approvals/{approvalID}/review", rest.AdaptHandler(editorialHandler.ReviewApproval))
	r.Get("/editorial/approvals", rest.AdaptHandler(editorialHandler.ListApprovals))

	r.Get("/editorial/calendar", rest.AdaptHandler(editorialHandler.ListCalendarEvents))
	r.Post("/editorial/calendar", rest.AdaptHandler(editorialHandler.CreateCalendarEvent))
	r.Put("/editorial/calendar/{id}", rest.AdaptHandler(editorialHandler.UpdateCalendarEvent))
	r.Delete("/editorial/calendar/{id}", rest.AdaptHandler(editorialHandler.DeleteCalendarEvent))

	r.Get("/editorial/widgets", rest.AdaptHandler(editorialHandler.ListWidgets))
	r.Put("/editorial/widgets/{id}", rest.AdaptHandler(editorialHandler.UpdateWidget))

	r.Get("/editorial/ai-insights", rest.AdaptHandler(editorialHandler.AIInsights))
}

func registerResearchRoutes(r chi.Router, deps *Dependencies) { //nolint:unused // kept for future route registration
	researchHandler := researchModule.NewHandler(deps.ResearchSvc, deps.Log)

	r.Get("/research", rest.AdaptHandler(researchHandler.ListJobs))
	r.Post("/research", rest.AdaptHandler(researchHandler.CreateJob))
	r.Get("/research/search", rest.AdaptHandler(researchHandler.SearchByTopic))
	r.Get("/research/{id}", rest.AdaptHandler(researchHandler.GetJob))
	r.Put("/research/{id}", rest.AdaptHandler(researchHandler.UpdateJob))
	r.Delete("/research/{id}", rest.AdaptHandler(researchHandler.DeleteJob))
	r.Get("/research/{id}/briefing", rest.AdaptHandler(researchHandler.GetBriefing))

	registerWriterRoutes(r, deps)
}

func registerWriterRoutes(r chi.Router, deps *Dependencies) { //nolint:unused // kept for future route registration
	writerHandler := writerModule.NewHandler(deps.WriterSvc, deps.Log)

	r.Get("/writer/styles", rest.AdaptHandler(writerHandler.ListStyles))
	r.Get("/writer", rest.AdaptHandler(writerHandler.ListJobs))
	r.Post("/writer", rest.AdaptHandler(writerHandler.CreateJob))
	r.Get("/writer/{id}", rest.AdaptHandler(writerHandler.GetJob))
	r.Put("/writer/{id}", rest.AdaptHandler(writerHandler.UpdateJob))
	r.Delete("/writer/{id}", rest.AdaptHandler(writerHandler.DeleteJob))

	r.Post("/writer/{id}/outline", rest.AdaptHandler(writerHandler.CreateOutline))
	r.Get("/writer/{id}/outline", rest.AdaptHandler(writerHandler.ListOutline))

	r.Post("/writer/{id}/sections", rest.AdaptHandler(writerHandler.CreateSection))
	r.Get("/writer/{id}/sections", rest.AdaptHandler(writerHandler.ListSections))
	r.Get("/writer/{id}/sections/{sectionID}", rest.AdaptHandler(writerHandler.GetSection))
	r.Put("/writer/{id}/sections/{sectionID}", rest.AdaptHandler(writerHandler.UpdateSection))

	r.Post("/writer/{id}/versions", rest.AdaptHandler(writerHandler.CreateVersion))
	r.Get("/writer/{id}/versions", rest.AdaptHandler(writerHandler.ListVersions))
	r.Get("/writer/{id}/versions/{versionID}", rest.AdaptHandler(writerHandler.GetVersion))
	r.Post("/writer/{id}/versions/{versionID}/restore", rest.AdaptHandler(writerHandler.RestoreVersion))
}

func registerEditorialEngineRoutes(r chi.Router, deps *Dependencies) {
	eeHandler := editorialEngineModule.NewHandler(deps.EditorialEngineSvc, deps.Log)

	r.Get("/editorial-engine/pipelines", rest.AdaptHandler(eeHandler.ListPipelines))
	r.Post("/editorial-engine/pipelines", rest.AdaptHandler(eeHandler.CreatePipeline))
	r.Get("/editorial-engine/pipelines/{id}", rest.AdaptHandler(eeHandler.GetPipeline))
	r.Put("/editorial-engine/pipelines/{id}", rest.AdaptHandler(eeHandler.UpdatePipeline))
	r.Get("/editorial-engine/pipelines/{id}/stages", rest.AdaptHandler(eeHandler.ListPipelineStages))
	r.Get("/editorial-engine/pipelines/{id}/stages/{stageID}", rest.AdaptHandler(eeHandler.GetPipelineStage))
	r.Put("/editorial-engine/pipelines/{id}/stages/{stageID}", rest.AdaptHandler(eeHandler.UpdatePipelineStage))

	r.Get("/editorial-engine/styles", rest.AdaptHandler(eeHandler.GetStyleRules))
	r.Put("/editorial-engine/styles", rest.AdaptHandler(eeHandler.UpsertStyleRules))

	r.Post("/editorial-engine/jobs/{id}/seo", rest.AdaptHandler(eeHandler.CreateSEOData))
	r.Get("/editorial-engine/jobs/{id}/seo", rest.AdaptHandler(eeHandler.GetSEOData))
	r.Put("/editorial-engine/jobs/{id}/seo", rest.AdaptHandler(eeHandler.UpdateSEOData))

	r.Post("/editorial-engine/jobs/{id}/quality", rest.AdaptHandler(eeHandler.CreateQualityScore))
	r.Get("/editorial-engine/jobs/{id}/quality", rest.AdaptHandler(eeHandler.GetQualityScore))
	r.Get("/editorial-engine/jobs/{id}/quality/history", rest.AdaptHandler(eeHandler.ListQualityScores))

	r.Post("/editorial-engine/jobs/{id}/translations", rest.AdaptHandler(eeHandler.CreateTranslation))
	r.Get("/editorial-engine/jobs/{id}/translations", rest.AdaptHandler(eeHandler.ListTranslations))
	r.Get("/editorial-engine/jobs/{id}/translations/{translationID}", rest.AdaptHandler(eeHandler.GetTranslation))
	r.Put("/editorial-engine/jobs/{id}/translations/{translationID}", rest.AdaptHandler(eeHandler.UpdateTranslation))

	r.Post("/editorial-engine/jobs/{id}/prompt", rest.AdaptHandler(eeHandler.CreatePromptData))
	r.Get("/editorial-engine/jobs/{id}/prompt", rest.AdaptHandler(eeHandler.GetPromptData))
}

func registerAIRoutes(r chi.Router, deps *Dependencies) {
	if deps.AIManager == nil {
		return
	}
	aiHandler := aiModule.NewHandler(deps.AIManager, deps.Log)

	r.Get("/ai/providers", rest.AdaptHandler(aiHandler.ListProviders))
	r.Get("/ai/health", rest.AdaptHandler(aiHandler.HealthCheck))
	r.Post("/ai/test", rest.AdaptHandler(aiHandler.TestProvider))
	r.Post("/ai/prompt", rest.AdaptHandler(aiHandler.PreviewPrompt))
	r.Get("/ai/capabilities", rest.AdaptHandler(aiHandler.GetCapabilities))
}

func registerGeneratorRoutes(r chi.Router, deps *Dependencies) {
	genHandler := generatorModule.NewHandler(deps.GeneratorSvc, deps.Log)

	r.Post("/generator", rest.AdaptHandler(genHandler.CreateJob))
	r.Get("/generator", rest.AdaptHandler(genHandler.ListJobs))
	r.Get("/generator/{id}/detail", rest.AdaptHandler(genHandler.GetJob))
	r.Get("/generator/{id}", rest.AdaptHandler(genHandler.GetJobSimple))
	r.Put("/generator/{id}", rest.AdaptHandler(genHandler.UpdateJob))

	r.Post("/generator/{id}/start", rest.AdaptHandler(genHandler.StartJob))
	r.Post("/generator/{id}/pause", rest.AdaptHandler(genHandler.PauseJob))
	r.Post("/generator/{id}/resume", rest.AdaptHandler(genHandler.ResumeJob))
	r.Post("/generator/{id}/cancel", rest.AdaptHandler(genHandler.CancelJob))
	r.Post("/generator/{id}/retry", rest.AdaptHandler(genHandler.RetryStage))
	r.Post("/generator/{id}/restart", rest.AdaptHandler(genHandler.RestartJob))

	r.Get("/generator/{id}/pipeline", rest.AdaptHandler(genHandler.GetPipeline))
	r.Get("/generator/{id}/logs", rest.AdaptHandler(genHandler.GetLogs))

	r.Post("/generator/{id}/quality", rest.AdaptHandler(genHandler.CheckQualityGate))
	r.Get("/generator/{id}/quality", rest.AdaptHandler(genHandler.GetQualityGates))

	r.Get("/generator/stats", rest.AdaptHandler(genHandler.GetStats))
	r.Get("/generator/dashboard", rest.AdaptHandler(genHandler.GetDashboard))

	r.Post("/generator/{id}/prompt", rest.AdaptHandler(genHandler.AssemblePrompt))
}

func registerAutocontentRoutes(r chi.Router, deps *Dependencies) {
	if deps.AutocontentSvc == nil {
		return
	}
	autocontentModule.RegisterRoutes(r, deps.AutocontentSvc, deps.Log)
}

func registerHumanWriterRoutes(r chi.Router, deps *Dependencies) {
	if deps.HumanWriterSvc == nil {
		return
	}
	humanwriterModule.RegisterRoutes(r, deps.HumanWriterSvc, deps.Log)
}

func registerArticlePipelineRoutes(r chi.Router, deps *Dependencies) {
	if deps.ArticlePipelineSvc == nil {
		return
	}
	articlepipelineModule.RegisterRoutes(r, deps.ArticlePipelineSvc, deps.Log)
}

func registerPublisherRoutes(r chi.Router, deps *Dependencies) {
	if deps.PublisherSvc == nil {
		return
	}
	publisherModule.RegisterRoutes(r, deps.PublisherSvc, deps.Log)
}

func registerSeoEngineRoutes(r chi.Router, deps *Dependencies) {
	if deps.SeoEngineSvc == nil {
		return
	}
	seoengineModule.RegisterRoutes(r, deps.SeoEngineSvc, deps.Log)
}

func wrapMiddleware(mw func(http.Handler) http.Handler, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mw(handler).ServeHTTP(w, r)
	}
}
