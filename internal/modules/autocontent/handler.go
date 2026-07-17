package autocontent

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
	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "topic is required")
		return
	}

	job, err := h.svc.CreateJob(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else {
			h.log.Error("failed to create autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create autocontent job")
		}
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else {
			h.log.Error("failed to get autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get autocontent job")
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
	step := ctx.Request.URL.Query().Get("step")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	jobs, err := h.svc.ListJobs(ctx.Request.Context(), siteID, status, language, step, limit, offset)
	if err != nil {
		h.log.Error("failed to list autocontent jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list autocontent jobs")
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to update autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update autocontent job")
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

	err = h.svc.DeleteJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else {
			h.log.Error("failed to delete autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete autocontent job")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else if errors.Is(err, ErrJobAlreadyRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already running")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to start autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to start autocontent job")
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
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else if errors.Is(err, ErrJobNotRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job is not running")
		} else {
			h.log.Error("failed to pause autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to pause autocontent job")
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
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else if errors.Is(err, ErrJobPaused) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job is not paused")
		} else {
			h.log.Error("failed to resume autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to resume autocontent job")
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

	reason := ctx.Request.URL.Query().Get("reason")
	if reason == "" {
		reason = "user requested cancellation"
	}

	job, err := h.svc.CancelJob(ctx.Request.Context(), siteID, jobID, reason)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to cancel autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel autocontent job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) RetryStep(ctx *rest.Context) {
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

	var req RetryStepRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.StepName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "step_name is required")
		return
	}

	job, err := h.svc.RetryStep(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else if errors.Is(err, ErrStepNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "step not found")
		} else if errors.Is(err, ErrMaxRetriesExceeded) {
			ctx.Error(http.StatusConflict, "CONFLICT", "max retries exceeded")
		} else {
			h.log.Error("failed to retry step", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to retry step")
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "autocontent job not found")
		} else {
			h.log.Error("failed to restart autocontent job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to restart autocontent job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) GetSteps(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	steps, err := h.svc.GetSteps(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to get steps", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get steps")
		return
	}

	ctx.JSON(http.StatusOK, steps)
}

func (h *Handler) SaveResult(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	var body struct {
		StepName string                 `json:"step_name"`
		Content  string                 `json:"content"`
		Summary  string                 `json:"summary"`
		Score    float64                `json:"score"`
		Passed   bool                   `json:"passed"`
		Data     map[string]interface{} `json:"data"`
	}
	if err := ctx.Decode(&body); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if body.StepName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "step_name is required")
		return
	}

	result, err := h.svc.SaveResult(ctx.Request.Context(), jobID, body.StepName, body.Content, body.Summary, body.Score, body.Passed, body.Data)
	if err != nil {
		h.log.Error("failed to save result", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to save result")
		return
	}

	ctx.JSON(http.StatusCreated, result)
}

func (h *Handler) GetResults(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	results, err := h.svc.GetResults(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to get results", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get results")
		return
	}

	ctx.JSON(http.StatusOK, results)
}

func (h *Handler) GetResultByStep(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	stepName := chi.URLParam(ctx.Request, "stepName")
	if stepName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "step name is required")
		return
	}

	result, err := h.svc.GetResultByStep(ctx.Request.Context(), jobID, stepName)
	if err != nil {
		if errors.Is(err, ErrResultNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "result not found")
		} else {
			h.log.Error("failed to get result", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get result")
		}
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) GetMetrics(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	metrics, err := h.svc.GetMetrics(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get metrics", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get metrics")
		return
	}

	ctx.JSON(http.StatusOK, metrics)
}

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

func (h *Handler) AddToQueue(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req QueueRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	item, err := h.svc.AddToQueue(ctx.Request.Context(), siteID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
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

func (h *Handler) UpdateQueueItem(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(ctx.Request, "queueID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid queue item ID")
		return
	}

	var req UpdateQueueRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	item, err := h.svc.UpdateQueueItem(ctx.Request.Context(), siteID, itemID, req)
	if err != nil {
		if errors.Is(err, ErrQueueItemNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "queue item not found")
		} else {
			h.log.Error("failed to update queue item", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update queue item")
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

func (h *Handler) CreateTemplate(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreateTemplateRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "template name is required")
		return
	}

	tmpl, err := h.svc.CreateTemplate(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to create template", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create template")
		return
	}

	ctx.JSON(http.StatusCreated, tmpl)
}

func (h *Handler) ListTemplates(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	templates, err := h.svc.ListTemplates(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list templates", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list templates")
		return
	}

	ctx.JSON(http.StatusOK, templates)
}
