package publisher

import (
	"errors"
	"net/http"
	"strconv"

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

func (h *Handler) Publish(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req PublishRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	resp, err := h.svc.PublishArticle(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrTitleRequired) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		} else if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else if errors.Is(err, ErrInvalidSlug) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid slug format")
		} else if errors.Is(err, ErrDuplicateSlug) {
			ctx.Error(http.StatusConflict, "CONFLICT", "duplicate slug for site")
		} else if errors.Is(err, ErrInvalidVisibility) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid visibility")
		} else {
			h.log.Error("failed to publish article", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to publish article")
		}
		return
	}

	ctx.JSON(http.StatusCreated, resp)
}

func (h *Handler) Schedule(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req ScheduleRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	resp, err := h.svc.SchedulePublication(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else {
			h.log.Error("failed to schedule publication", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to schedule publication")
		}
		return
	}

	ctx.JSON(http.StatusCreated, resp)
}

func (h *Handler) Update(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	var req UpdatePublicationRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	pub, err := h.svc.UpdatePublication(ctx.Request.Context(), siteID, userID, pubID, req)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else if errors.Is(err, ErrDuplicateSlug) {
			ctx.Error(http.StatusConflict, "CONFLICT", "duplicate slug for site")
		} else if errors.Is(err, ErrInvalidSlug) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid slug format")
		} else if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else if errors.Is(err, ErrTitleRequired) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		} else {
			h.log.Error("failed to update publication", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update publication")
		}
		return
	}

	ctx.JSON(http.StatusOK, pub)
}

func (h *Handler) Unpublish(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	reason := ctx.Request.URL.Query().Get("reason")

	pub, err := h.svc.Unpublish(ctx.Request.Context(), siteID, userID, pubID, reason)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else if errors.Is(err, ErrPublicationNotPublished) {
			ctx.Error(http.StatusConflict, "CONFLICT", "publication is not published")
		} else {
			h.log.Error("failed to unpublish", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to unpublish")
		}
		return
	}

	ctx.JSON(http.StatusOK, pub)
}

func (h *Handler) Republish(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	pub, err := h.svc.Republish(ctx.Request.Context(), siteID, userID, pubID)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else if errors.Is(err, ErrPublicationAlreadyPublished) {
			ctx.Error(http.StatusConflict, "CONFLICT", "publication is already published")
		} else {
			h.log.Error("failed to republish", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to republish")
		}
		return
	}

	ctx.JSON(http.StatusOK, pub)
}

func (h *Handler) CancelSchedule(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	scheduleID, err := uuid.Parse(chi.URLParam(ctx.Request, "scheduleID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid schedule ID")
		return
	}

	reason := ctx.Request.URL.Query().Get("reason")

	sched, err := h.svc.CancelSchedule(ctx.Request.Context(), siteID, userID, scheduleID, reason)
	if err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "schedule not found")
		} else {
			h.log.Error("failed to cancel schedule", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel schedule")
		}
		return
	}

	ctx.JSON(http.StatusOK, sched)
}

func (h *Handler) GetPublication(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	pub, err := h.svc.GetPublication(ctx.Request.Context(), siteID, pubID)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else {
			h.log.Error("failed to get publication", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get publication")
		}
		return
	}

	ctx.JSON(http.StatusOK, pub)
}

func (h *Handler) ListPublications(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	language := ctx.Request.URL.Query().Get("language")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	pubs, total, err := h.svc.ListPublications(ctx.Request.Context(), siteID, status, language, limit, offset)
	if err != nil {
		h.log.Error("failed to list publications", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list publications")
		return
	}

	ctx.JSON(http.StatusOK, PublicationListResponse{Publications: pubs, Total: total})
}

func (h *Handler) DeletePublication(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	err = h.svc.DeletePublication(ctx.Request.Context(), siteID, userID, pubID)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else {
			h.log.Error("failed to delete publication", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete publication")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

func (h *Handler) GetHistory(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	history, err := h.svc.GetPublicationHistory(ctx.Request.Context(), siteID, pubID, limit, offset)
	if err != nil {
		h.log.Error("failed to get history", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get history")
		return
	}

	ctx.JSON(http.StatusOK, history)
}

func (h *Handler) GetMetrics(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}

	metrics, err := h.svc.GetPublicationMetrics(ctx.Request.Context(), siteID, pubID)
	if err != nil {
		if errors.Is(err, ErrMetricsNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "metrics not found")
		} else {
			h.log.Error("failed to get metrics", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get metrics")
		}
		return
	}

	ctx.JSON(http.StatusOK, metrics)
}

func (h *Handler) GetMetricsSummary(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	summary, err := h.svc.GetMetricsSummary(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get metrics summary", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get metrics summary")
		return
	}

	ctx.JSON(http.StatusOK, summary)
}

// --- Queue ---

func (h *Handler) AddToQueue(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req QueueRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	item, err := h.svc.AddToQueue(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrPublicationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "publication not found")
		} else if errors.Is(err, ErrInvalidAction) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid queue action")
		} else {
			h.log.Error("failed to add to queue", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to add to queue")
		}
		return
	}

	ctx.JSON(http.StatusCreated, item)
}

func (h *Handler) ListQueue(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	items, err := h.svc.ListQueue(ctx.Request.Context(), siteID, status, limit, offset)
	if err != nil {
		h.log.Error("failed to list queue", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list queue")
		return
	}

	ctx.JSON(http.StatusOK, items)
}

func (h *Handler) RetryQueue(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(ctx.Request, "itemID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid queue item ID")
		return
	}

	item, err := h.svc.RetryQueueItem(ctx.Request.Context(), siteID, userID, itemID)
	if err != nil {
		if errors.Is(err, ErrQueueItemNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "queue item not found")
		} else if errors.Is(err, ErrMaxRetriesExceeded) {
			ctx.Error(http.StatusConflict, "CONFLICT", "max retries exceeded")
		} else {
			h.log.Error("failed to retry queue item", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to retry queue item")
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

// --- Schedules ---

func (h *Handler) ListSchedules(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	schedules, err := h.svc.ListSchedules(ctx.Request.Context(), siteID, status, limit, offset)
	if err != nil {
		h.log.Error("failed to list schedules", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list schedules")
		return
	}

	ctx.JSON(http.StatusOK, schedules)
}

func (h *Handler) GetSchedule(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	scheduleID, err := uuid.Parse(chi.URLParam(ctx.Request, "scheduleID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid schedule ID")
		return
	}

	sched, err := h.svc.GetSchedule(ctx.Request.Context(), siteID, scheduleID)
	if err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "schedule not found")
		} else {
			h.log.Error("failed to get schedule", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get schedule")
		}
		return
	}

	ctx.JSON(http.StatusOK, sched)
}

// --- Validation ---

func (h *Handler) ValidateSlug(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	slug := ctx.Request.URL.Query().Get("slug")
	if slug == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "slug is required")
		return
	}

	available, validSlug, err := h.svc.ValidateSlug(ctx.Request.Context(), siteID, slug)
	if err != nil {
		if errors.Is(err, ErrInvalidSlug) {
			ctx.JSON(http.StatusOK, map[string]interface{}{
				"valid":     false,
				"slug":      slug,
				"available": false,
				"error":     "invalid slug format",
			})
		} else {
			h.log.Error("failed to validate slug", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to validate slug")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"valid":     true,
		"slug":      validSlug,
		"available": available,
	})
}

func (h *Handler) GenerateSlug(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	title := ctx.Request.URL.Query().Get("title")
	if title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	slug, err := h.svc.GenerateSlug(ctx.Request.Context(), siteID, title)
	if err != nil {
		h.log.Error("failed to generate slug", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to generate slug")
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"slug": slug,
		"url":  h.svc.val.GenerateURL(slug, "pt", "https://example.com"),
	})
}
