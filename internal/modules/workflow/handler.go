package workflow

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

// --- Dashboard ---

func (h *Handler) GetDashboard(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	dash, err := h.svc.GetDashboard(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get dashboard", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get dashboard")
		return
	}

	ctx.JSON(http.StatusOK, dash)
}

func (h *Handler) RefreshDashboard(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	dash, err := h.svc.RefreshDashboard(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to refresh dashboard", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to refresh dashboard")
		return
	}

	ctx.JSON(http.StatusOK, dash)
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
	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	job, err := h.svc.CreateJob(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else {
			h.log.Error("failed to create workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create workflow job")
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
		} else {
			h.log.Error("failed to get workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get workflow job")
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
		h.log.Error("failed to list workflow jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list workflow jobs")
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else if errors.Is(err, ErrInvalidPriority) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "priority must be 1-10")
		} else {
			h.log.Error("failed to update workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update workflow job")
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
		} else {
			h.log.Error("failed to delete workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete workflow job")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// --- Workflow Control ---

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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
		} else if errors.Is(err, ErrJobAlreadyRunning) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already running")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to start workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to start workflow job")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) PauseJob(ctx *rest.Context) {
	h.handleJobTransition(ctx, "pause",
		h.svc.PauseJob,
		ErrJobNotRunning, "job is not running",
	)
}

func (h *Handler) ResumeJob(ctx *rest.Context) {
	h.handleJobTransition(ctx, "resume",
		h.svc.ResumeJob,
		ErrJobPaused, "job is not paused",
	)
}

type jobTransitionFunc func(ctx context.Context, siteID, jobID uuid.UUID) (*WorkflowJob, error)

func (h *Handler) handleJobTransition(ctx *rest.Context, verb string, fn jobTransitionFunc, statusErr error, statusMsg string) {
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

	job, err := fn(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
		} else if errors.Is(err, statusErr) {
			ctx.Error(http.StatusConflict, "CONFLICT", statusMsg)
		} else {
			h.log.Error("failed to "+verb+" workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to "+verb+" workflow job")
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
		} else if errors.Is(err, ErrJobAlreadyCompleted) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already completed")
		} else if errors.Is(err, ErrJobAlreadyCancelled) {
			ctx.Error(http.StatusConflict, "CONFLICT", "job already cancelled")
		} else {
			h.log.Error("failed to cancel workflow job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel workflow job")
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

	var req RetryRequest
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
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "workflow job not found")
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

// --- Steps ---

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

func (h *Handler) AdvanceStep(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	var body struct {
		StepName string                 `json:"step_name"`
		Status   StepStatus             `json:"status"`
		Progress float64                `json:"progress"`
		Metadata map[string]interface{} `json:"metadata"`
		Error    string                 `json:"error"`
		Duration int64                  `json:"duration_ms"`
	}
	if err := ctx.Decode(&body); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if body.StepName == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "step_name is required")
		return
	}

	step, err := h.svc.AdvanceStep(ctx.Request.Context(), jobID, body.StepName, body.Status, body.Progress, body.Metadata, body.Error, body.Duration)
	if err != nil {
		h.log.Error("failed to advance step", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to advance step")
		return
	}

	ctx.JSON(http.StatusOK, step)
}

// --- Queue ---

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

func (h *Handler) PauseQueue(ctx *rest.Context) {
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

	item, err := h.svc.PauseQueue(ctx.Request.Context(), siteID, itemID)
	if err != nil {
		if errors.Is(err, ErrQueueItemNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "queue item not found")
		} else {
			h.log.Error("failed to pause queue", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to pause queue")
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

func (h *Handler) ResumeQueue(ctx *rest.Context) {
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

	item, err := h.svc.ResumeQueue(ctx.Request.Context(), siteID, itemID)
	if err != nil {
		if errors.Is(err, ErrQueueItemNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "queue item not found")
		} else {
			h.log.Error("failed to resume queue", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to resume queue")
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

func (h *Handler) CancelQueue(ctx *rest.Context) {
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

	item, err := h.svc.CancelQueue(ctx.Request.Context(), siteID, itemID)
	if err != nil {
		if errors.Is(err, ErrQueueItemNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "queue item not found")
		} else {
			h.log.Error("failed to cancel queue", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel queue")
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

// --- Notifications ---

func (h *Handler) ListNotifications(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	notifType := ctx.Request.URL.Query().Get("type")
	unreadOnly := ctx.Request.URL.Query().Get("unread") == "true"
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	result, err := h.svc.ListNotifications(ctx.Request.Context(), siteID, notifType, unreadOnly, limit, offset)
	if err != nil {
		h.log.Error("failed to list notifications", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list notifications")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) MarkNotificationRead(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	notifID, err := uuid.Parse(chi.URLParam(ctx.Request, "notifID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid notification ID")
		return
	}

	err = h.svc.MarkNotificationRead(ctx.Request.Context(), siteID, notifID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "notification not found")
		} else {
			h.log.Error("failed to mark notification read", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to mark notification read")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) MarkAllNotificationsRead(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	err := h.svc.MarkAllNotificationsRead(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to mark all notifications read", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to mark notifications read")
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// --- Metrics & Stats ---

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

// --- History ---

func (h *Handler) ListHistory(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var jobID *uuid.UUID
	if jid := ctx.Request.URL.Query().Get("job_id"); jid != "" {
		parsed, err := uuid.Parse(jid)
		if err == nil {
			jobID = &parsed
		}
	}
	action := ctx.Request.URL.Query().Get("action")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	entries, err := h.svc.ListHistory(ctx.Request.Context(), siteID, jobID, action, limit, offset)
	if err != nil {
		h.log.Error("failed to list history", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list history")
		return
	}

	ctx.JSON(http.StatusOK, entries)
}

// --- Automation ---

func (h *Handler) ExecuteAction(ctx *rest.Context) {
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

	var action AutomationAction
	if err := ctx.Decode(&action); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if action.Action == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "action is required")
		return
	}

	job, err := h.svc.ExecuteAction(ctx.Request.Context(), siteID, userID, action)
	if err != nil {
		if errors.Is(err, ErrInvalidAction) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid automation action")
		} else if errors.Is(err, ErrInvalidTitle) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		} else if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "job not found")
		} else {
			h.log.Error("failed to execute action", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to execute action")
		}
		return
	}

	ctx.JSON(http.StatusOK, job)
}

// --- Queue Process ---

func (h *Handler) ProcessQueue(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	item, err := h.svc.ProcessQueue(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to process queue", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to process queue")
		return
	}

	ctx.JSON(http.StatusOK, item)
}
