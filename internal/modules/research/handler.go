package research

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

func (h *Handler) CreateJob(ctx *rest.Context) {
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

	var req CreateResearchJobRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "topic is required")
		return
	}

	if req.Language != "pt" && req.Language != "en" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		return
	}

	job, err := h.svc.CreateJob(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		h.log.Error("failed to create research job", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create research job")
		return
	}

	ctx.JSON(http.StatusCreated, job)
}

func (h *Handler) GetJob(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	job, err := h.svc.GetJobDetail(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrResearchJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "research job not found")
		} else {
			h.log.Error("failed to get research job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get research job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) ListJobs(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := JobStatus(ctx.Request.URL.Query().Get("status"))

	jobs, err := h.svc.ListJobs(ctx.Request.Context(), siteID, status)
	if err != nil {
		h.log.Error("failed to list research jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list research jobs")
		return
	}

	ctx.JSON(http.StatusOK, jobs)
}

func (h *Handler) SearchByTopic(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	query := ctx.Request.URL.Query().Get("q")
	if query == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_QUERY", "search query is required")
		return
	}

	jobs, err := h.svc.SearchByTopic(ctx.Request.Context(), siteID, query)
	if err != nil {
		h.log.Error("failed to search research jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to search research jobs")
		return
	}

	ctx.JSON(http.StatusOK, jobs)
}

func (h *Handler) GetBriefing(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	briefing, err := h.svc.GetBriefing(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrResearchJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "research job not found")
		} else if errors.Is(err, ErrBriefingNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "briefing not found")
		} else {
			h.log.Error("failed to get briefing", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get briefing")
		}
		return
	}

	ctx.JSON(http.StatusOK, briefing)
}

func (h *Handler) UpdateJob(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	var req UpdateResearchJobRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	job, err := h.svc.UpdateJob(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		if errors.Is(err, ErrResearchJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "research job not found")
		} else {
			h.log.Error("failed to update research job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update research job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) DeleteJob(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	if err := h.svc.DeleteJob(ctx.Request.Context(), siteID, jobID); err != nil {
		if errors.Is(err, ErrResearchJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "research job not found")
		} else {
			h.log.Error("failed to delete research job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete research job")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
