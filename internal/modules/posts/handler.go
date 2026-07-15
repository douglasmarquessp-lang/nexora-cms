package posts

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/modules/auth"
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

	authorID, ok := auth.GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req CreatePostRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	post, err := h.svc.Create(ctx.Request.Context(), siteID, authorID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPostStatus):
			ctx.Error(http.StatusBadRequest, "INVALID_STATUS", err.Error())
		case errors.Is(err, ErrPostSlugExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to create post", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create post")
		}
		return
	}

	ctx.JSON(http.StatusCreated, post)
}

func (h *Handler) Get(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	post, err := h.svc.GetByID(ctx.Request.Context(), siteID, postID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPostNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "post not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get post", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get post")
		}
		return
	}

	ctx.JSON(http.StatusOK, post)
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

	post, err := h.svc.GetBySlug(ctx.Request.Context(), siteID, slug)
	if err != nil {
		switch {
		case errors.Is(err, ErrPostNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "post not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to get post by slug", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get post")
		}
		return
	}

	ctx.JSON(http.StatusOK, post)
}

func (h *Handler) List(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	q := ctx.Request.URL.Query()

	statusStr := q.Get("status")
	var status PostStatus
	if statusStr != "" {
		status = PostStatus(statusStr)
	}

	authorIDStr := q.Get("author_id")
	var authorID uuid.UUID
	if authorIDStr != "" {
		authorID, _ = uuid.Parse(authorIDStr)
	}

	categoryIDStr := q.Get("category_id")
	var categoryID uuid.UUID
	if categoryIDStr != "" {
		categoryID, _ = uuid.Parse(categoryIDStr)
	}

	search := q.Get("search")
	sort := q.Get("sort")
	order := q.Get("order")

	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	resp, err := h.svc.List(ctx.Request.Context(), PostListRequest{
		SiteID:     siteID,
		Status:     status,
		AuthorID:   authorID,
		CategoryID: categoryID,
		Search:     search,
		Page:       page,
		PerPage:    perPage,
		Sort:       sort,
		Order:      order,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPagination):
			ctx.Error(http.StatusBadRequest, "INVALID_PAGINATION", err.Error())
		case errors.Is(err, ErrInvalidPostStatus):
			ctx.Error(http.StatusBadRequest, "INVALID_STATUS", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to list posts", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list posts")
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

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	var req UpdatePostRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	post, err := h.svc.Update(ctx.Request.Context(), siteID, postID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPostNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "post not found")
		case errors.Is(err, ErrInvalidPostStatus):
			ctx.Error(http.StatusBadRequest, "INVALID_STATUS", err.Error())
		case errors.Is(err, ErrPostSlugExists):
			ctx.Error(http.StatusConflict, "SLUG_EXISTS", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to update post", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update post")
		}
		return
	}

	ctx.JSON(http.StatusOK, post)
}

func (h *Handler) Delete(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	err = h.svc.Delete(ctx.Request.Context(), siteID, postID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPostNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "post not found")
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to delete post", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete post")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SetStatus(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	var req SetStatusRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if strings.TrimSpace(string(req.Status)) == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "status is required")
		return
	}

	err = h.svc.SetStatus(ctx.Request.Context(), siteID, postID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrPostNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "post not found")
		case errors.Is(err, ErrInvalidPostStatus):
			ctx.Error(http.StatusBadRequest, "INVALID_STATUS", err.Error())
		case errors.Is(err, ErrDatabaseNotAvail):
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", err.Error())
		default:
			h.log.Error("failed to set post status", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to set post status")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
