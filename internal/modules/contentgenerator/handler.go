package contentgenerator

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

// --- Jobs ---

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

	var req CreateJobRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Language != "pt" && req.Language != "en" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		return
	}

	job, err := h.svc.CreateJob(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		h.log.Error("failed to create generation job", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create generation job")
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
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "generation job not found")
		} else {
			h.log.Error("failed to get generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) GetJobSimple(ctx *rest.Context) {
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

	job, err := h.svc.GetJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "generation job not found")
		} else {
			h.log.Error("failed to get generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get generation job")
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

	status := ctx.Request.URL.Query().Get("status")
	language := ctx.Request.URL.Query().Get("language")
	stage := ctx.Request.URL.Query().Get("stage")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	jobs, err := h.svc.ListJobs(ctx.Request.Context(), siteID, status, language, stage, limit, offset)
	if err != nil {
		h.log.Error("failed to list generation jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list generation jobs")
		return
	}

	ctx.JSON(http.StatusOK, jobs)
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

	var req UpdateJobRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	job, err := h.svc.UpdateJob(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "generation job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to update generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

// --- Pipeline Control ---

func (h *Handler) StartJob(ctx *rest.Context) {
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

	job, err := h.svc.StartJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "generation job not found")
		} else if errors.Is(err, ErrJobAlreadyRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already running")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to start generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to start generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) PauseJob(ctx *rest.Context) {
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

	job, err := h.svc.PauseJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job is not running")
		} else {
			h.log.Error("failed to pause generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to pause generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) ResumeJob(ctx *rest.Context) {
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

	job, err := h.svc.ResumeJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job is not paused")
		} else {
			h.log.Error("failed to resume generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to resume generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) CancelJob(ctx *rest.Context) {
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

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ctx.Decode(&req); err != nil {
		req.Reason = "cancelled by user"
	}

	job, err := h.svc.CancelJob(ctx.Request.Context(), siteID, jobID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "generation job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to cancel generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) RetryStage(ctx *rest.Context) {
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

	stage := ctx.Request.URL.Query().Get("stage")
	if stage == "" {
		ctx.Error(http.StatusBadRequest, "MISSING_STAGE", "stage query parameter required")
		return
	}

	job, err := h.svc.RetryStage(ctx.Request.Context(), siteID, jobID, stage)
	if err != nil {
		if errors.Is(err, ErrMaxRetriesExceeded) {
			ctx.Error(http.StatusConflict, "MAX_RETRIES", "maximum retries exceeded")
		} else {
			h.log.Error("failed to retry stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to retry stage")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) RestartJob(ctx *rest.Context) {
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

	job, err := h.svc.RestartJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "generation job not found")
		} else {
			h.log.Error("failed to restart generation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to restart generation job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

// --- Pipeline ---

func (h *Handler) GetPipeline(ctx *rest.Context) {
	_, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	pipeline, err := h.svc.ListPipeline(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to list pipeline", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list pipeline")
		return
	}

	ctx.JSON(http.StatusOK, pipeline)
}

// --- Logs ---

func (h *Handler) GetLogs(ctx *rest.Context) {
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

	stage := ctx.Request.URL.Query().Get("stage")
	level := ctx.Request.URL.Query().Get("level")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	logs, err := h.svc.ListLogs(ctx.Request.Context(), jobID, stage, level, limit, offset)
	if err != nil {
		h.log.Error("failed to list logs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list logs")
		return
	}

	_ = siteID
	ctx.JSON(http.StatusOK, logs)
}

// --- Quality Gates ---

func (h *Handler) CheckQualityGate(ctx *rest.Context) {
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

	var req QualityGateRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	gate, err := h.svc.CheckQualityGate(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		h.log.Error("failed to check quality gate", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to check quality gate")
		return
	}

	ctx.JSON(http.StatusCreated, gate)
}

func (h *Handler) GetQualityGates(ctx *rest.Context) {
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

	gates, err := h.svc.GetQualityGates(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to get quality gates", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get quality gates")
		return
	}

	_ = siteID
	ctx.JSON(http.StatusOK, gates)
}

// --- Stats & Dashboard ---

func (h *Handler) GetStats(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	stats, err := h.svc.GetStats(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get stats", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get stats")
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

func (h *Handler) GetDashboard(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	stats, err := h.svc.GetDashboardStats(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get dashboard stats", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get dashboard stats")
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

// --- Prompt Assembly ---

func (h *Handler) AssemblePrompt(ctx *rest.Context) {
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

	prompt, err := h.svc.AssemblePrompt(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		h.log.Error("failed to assemble prompt", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to assemble prompt")
		return
	}

	ctx.JSON(http.StatusOK, prompt)
}
