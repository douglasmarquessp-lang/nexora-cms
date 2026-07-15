package tags

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/pkg/logger"
)

type Handler struct {
	svc *Service
	log *logger.Logger
}

func NewHandler(svc *Service, log *logger.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Create(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreateTagRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "name is required")
		return
	}

	tag, err := h.svc.Create(ctx.Request.Context(), siteID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrTagSlugExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to create tag", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create tag")
		}
		return
	}

	ctx.JSON(http.StatusCreated, tag)
}

func (h *Handler) Get(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid tag ID")
		return
	}

	tag, err := h.svc.GetByID(ctx.Request.Context(), siteID, tagID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTagNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "tag not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get tag", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get tag")
		}
		return
	}

	ctx.JSON(http.StatusOK, tag)
}

func (h *Handler) GetBySlug(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	slug := ctx.Request.URL.Query().Get("slug")
	if slug == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "slug query parameter is required")
		return
	}

	tag, err := h.svc.GetBySlug(ctx.Request.Context(), siteID, slug)
	if err != nil {
		switch {
		case errors.Is(err, ErrTagNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "tag not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get tag by slug", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get tag")
		}
		return
	}

	ctx.JSON(http.StatusOK, tag)
}

func (h *Handler) List(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	resp, err := h.svc.List(ctx.Request.Context(), siteID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to list tags", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list tags")
		}
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Update(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid tag ID")
		return
	}

	var req UpdateTagRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	tag, err := h.svc.Update(ctx.Request.Context(), siteID, tagID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrTagNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "tag not found")
		case errors.Is(err, ErrTagSlugExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to update tag", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update tag")
		}
		return
	}

	ctx.JSON(http.StatusOK, tag)
}

func (h *Handler) Delete(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	tagID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid tag ID")
		return
	}

	err = h.svc.Delete(ctx.Request.Context(), siteID, tagID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTagNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "tag not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to delete tag", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete tag")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
