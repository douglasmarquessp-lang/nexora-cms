package plugins

import (
	"github.com/go-chi/chi/v5"

	"nexora/internal/api/rest"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/plugins", rest.AdaptHandler(h.List))
	r.Get("/plugins/{id}", rest.AdaptHandler(h.Get))
	r.Post("/plugins/install", rest.AdaptHandler(h.Install))
	r.Post("/plugins/activate", rest.AdaptHandler(h.Activate))
	r.Post("/plugins/deactivate", rest.AdaptHandler(h.Deactivate))
	r.Post("/plugins/{id}/update", rest.AdaptHandler(h.Update))
	r.Delete("/plugins/{id}", rest.AdaptHandler(h.Delete))
	r.Get("/plugins/{id}/settings", rest.AdaptHandler(h.GetSettings))
	r.Patch("/plugins/{id}/settings", rest.AdaptHandler(h.UpdateSettings))
}
