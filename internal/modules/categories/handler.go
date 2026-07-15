package categories

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

	var req CreateCategoryRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "name is required")
		return
	}

	cat, err := h.svc.Create(ctx.Request.Context(), siteID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrCategorySlugExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		case errors.Is(err, ErrInvalidParentCategory):
			ctx.Error(http.StatusBadRequest, "INVALID_PARENT", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to create category", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create category")
		}
		return
	}

	ctx.JSON(http.StatusCreated, cat)
}

func (h *Handler) Get(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	catID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid category ID")
		return
	}

	cat, err := h.svc.GetByID(ctx.Request.Context(), siteID, catID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCategoryNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "category not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get category", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get category")
		}
		return
	}

	ctx.JSON(http.StatusOK, cat)
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
			h.log.Error("failed to list categories", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list categories")
		}
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Tree(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	tree, err := h.svc.Tree(ctx.Request.Context(), siteID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get category tree", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get category tree")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{"categories": tree})
}

func (h *Handler) Update(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	catID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid category ID")
		return
	}

	var req UpdateCategoryRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	cat, err := h.svc.Update(ctx.Request.Context(), siteID, catID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrCategoryNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "category not found")
		case errors.Is(err, ErrCategorySlugExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		case errors.Is(err, ErrInvalidParentCategory):
			ctx.Error(http.StatusBadRequest, "INVALID_PARENT", err.Error())
		case errors.Is(err, ErrCircularParent):
			ctx.Error(http.StatusBadRequest, "CIRCULAR_PARENT", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to update category", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update category")
		}
		return
	}

	ctx.JSON(http.StatusOK, cat)
}

func (h *Handler) Delete(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	catID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid category ID")
		return
	}

	err = h.svc.Delete(ctx.Request.Context(), siteID, catID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCategoryNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "category not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to delete category", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete category")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
