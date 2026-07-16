package media

import (
	"github.com/go-chi/chi/v5"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	casbinPkg "nexora/internal/pkg/casbin"
)

func RegisterRoutes(r chi.Router, h *Handler, enf *casbinPkg.Enforcer) {
	r.With(middleware.RequirePermission(enf, "media", "create")).Post("/media/upload", rest.AdaptHandler(h.Upload))
	r.With(middleware.RequirePermission(enf, "media", "read")).Get("/media", rest.AdaptHandler(h.List))
	r.With(middleware.RequirePermission(enf, "media", "read")).Get("/media/search", rest.AdaptHandler(h.Search))
	r.With(middleware.RequirePermission(enf, "folders", "read")).Get("/media/folders", rest.AdaptHandler(h.ListFolders))
	r.With(middleware.RequirePermission(enf, "folders", "create")).Post("/media/folders", rest.AdaptHandler(h.CreateFolder))
	r.With(middleware.RequirePermission(enf, "media", "read")).Get("/media/{id}", rest.AdaptHandler(h.Get))
	r.With(middleware.RequirePermission(enf, "media", "write")).Patch("/media/{id}", rest.AdaptHandler(h.Update))
	r.With(middleware.RequirePermission(enf, "media", "delete")).Delete("/media/{id}", rest.AdaptHandler(h.Delete))
	r.With(middleware.RequirePermission(enf, "media", "write")).Post("/media/{id}/restore", rest.AdaptHandler(h.Restore))
	r.With(middleware.RequirePermission(enf, "media", "write")).Post("/media/move", rest.AdaptHandler(h.Move))
	r.With(middleware.RequirePermission(enf, "media", "write")).Post("/media/copy", rest.AdaptHandler(h.Copy))
	r.With(middleware.RequirePermission(enf, "folders", "write")).Patch("/media/folders/{id}", rest.AdaptHandler(h.UpdateFolder))
	r.With(middleware.RequirePermission(enf, "folders", "delete")).Delete("/media/folders/{id}", rest.AdaptHandler(h.DeleteFolder))
}
