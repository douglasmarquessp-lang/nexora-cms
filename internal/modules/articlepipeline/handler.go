package articlepipeline

import (
	"context"
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

func (h *Handler) CreatePipeline(ctx *rest.Context) {
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

	var req CreatePipelineRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	job, err := h.svc.CreatePipeline(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		} else if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else if errors.Is(err, ErrInvalidPriority) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "priority must be between 1 and 10")
		} else {
			h.log.Error("failed to create pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create pipeline")
		}
		return
	}

	ctx.JSON(http.StatusCreated, job)
}

func (h *Handler) GetPipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	job, err := h.svc.GetPipelineDetail(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else {
			h.log.Error("failed to get pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get pipeline")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) ListPipelines(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	language := ctx.Request.URL.Query().Get("language")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	jobs, err := h.svc.ListPipelines(ctx.Request.Context(), siteID, status, language, limit, offset)
	if err != nil {
		h.log.Error("failed to list pipelines", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list pipelines")
		return
	}

	ctx.JSON(http.StatusOK, jobs)
}

func (h *Handler) UpdatePipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	var req UpdatePipelineRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	job, err := h.svc.UpdatePipeline(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else {
			h.log.Error("failed to update pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update pipeline")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) DeletePipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	err = h.svc.DeletePipeline(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else {
			h.log.Error("failed to delete pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete pipeline")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

func (h *Handler) StartPipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	job, err := h.svc.StartPipeline(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else if errors.Is(err, ErrJobAlreadyRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "pipeline already running")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "pipeline already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "pipeline already cancelled")
		} else {
			h.log.Error("failed to start pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to start pipeline")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) setPipelineStatus(ctx *rest.Context, action func(context.Context, uuid.UUID, uuid.UUID) (*PipelineJob, error), conflictErr error, conflictMsg, logMsg string) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	job, err := action(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else if errors.Is(err, conflictErr) {
			ctx.Error(http.StatusConflict, "CONFLICT", conflictMsg)
		} else {
			h.log.Error(logMsg, "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", logMsg)
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) PausePipeline(ctx *rest.Context) {
	h.setPipelineStatus(ctx,
		h.svc.PausePipeline,
		ErrJobNotRunning, "pipeline is not running", "failed to pause pipeline")
}

func (h *Handler) ResumePipeline(ctx *rest.Context) {
	h.setPipelineStatus(ctx,
		h.svc.ResumePipeline,
		ErrJobNotPaused, "pipeline is not paused", "failed to resume pipeline")
}

func (h *Handler) CancelPipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	reason := ctx.Request.URL.Query().Get("reason")
	if reason == "" {
		reason = "user requested cancellation"
	}

	job, err := h.svc.CancelPipeline(ctx.Request.Context(), siteID, jobID, reason)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "pipeline already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "pipeline already cancelled")
		} else {
			h.log.Error("failed to cancel pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel pipeline")
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
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	var body struct {
		StageName string `json:"stage_name"`
	}
	if err := ctx.Decode(&body); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if body.StageName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "stage_name is required")
		return
	}

	job, err := h.svc.RetryStage(ctx.Request.Context(), siteID, jobID, body.StageName)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else if errors.Is(err, ErrStageNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "stage not found")
		} else if errors.Is(err, ErrMaxRetriesExceeded) {
			ctx.Error(http.StatusConflict, "CONFLICT", "max retries exceeded")
		} else {
			h.log.Error("failed to retry stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to retry stage")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) RestartPipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	job, err := h.svc.RestartPipeline(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else {
			h.log.Error("failed to restart pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to restart pipeline")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) GetPipelineStages(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	steps, err := h.svc.GetPipelineStages(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to get stages", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get stages")
		return
	}

	ctx.JSON(http.StatusOK, steps)
}

func (h *Handler) UpdateStage(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	stageName := chi.URLParam(ctx.Request, "stageName")
	if stageName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "stage name is required")
		return
	}

	var req UpdateStageRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	job, err := h.svc.UpdateStage(ctx.Request.Context(), siteID, jobID, stageName, req)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else if errors.Is(err, ErrStageNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "stage not found")
		} else if errors.Is(err, ErrStageAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "stage already completed")
		} else {
			h.log.Error("failed to update stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update stage")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) RecordMetric(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	var req CreateMetricRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.MetricName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "metric_name is required")
		return
	}

	metric, err := h.svc.RecordMetric(ctx.Request.Context(), jobID, req)
	if err != nil {
		h.log.Error("failed to record metric", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to record metric")
		return
	}

	ctx.JSON(http.StatusCreated, metric)
}

func (h *Handler) GetPipelineMetrics(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	metrics, err := h.svc.GetPipelineMetrics(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to get metrics", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get metrics")
		return
	}

	ctx.JSON(http.StatusOK, metrics)
}

func (h *Handler) CreateQualityReport(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	var req CreateQualityReportRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.StageName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "stage_name is required")
		return
	}

	report, err := h.svc.CreateQualityReport(ctx.Request.Context(), jobID, req)
	if err != nil {
		h.log.Error("failed to create quality report", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create quality report")
		return
	}

	ctx.JSON(http.StatusCreated, report)
}

func (h *Handler) GetQualityReports(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	reports, err := h.svc.GetQualityReports(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to get quality reports", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get quality reports")
		return
	}

	ctx.JSON(http.StatusOK, reports)
}

func (h *Handler) CreateCandidate(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline job ID")
		return
	}

	var req CreateCandidateRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	candidate, err := h.svc.CreateCandidate(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		} else if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline job not found")
		} else {
			h.log.Error("failed to create candidate", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create publication candidate")
		}
		return
	}

	ctx.JSON(http.StatusCreated, candidate)
}

func (h *Handler) ListCandidates(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	candidates, err := h.svc.ListCandidates(ctx.Request.Context(), siteID, status, limit, offset)
	if err != nil {
		h.log.Error("failed to list candidates", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list candidates")
		return
	}

	ctx.JSON(http.StatusOK, candidates)
}

func (h *Handler) GetPipelineStats(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	stats, err := h.svc.GetPipelineStats(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get stats", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get pipeline stats")
		return
	}

	ctx.JSON(http.StatusOK, stats)
}
