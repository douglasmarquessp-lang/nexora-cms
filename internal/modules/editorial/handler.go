package editorial

import (
	"errors"
	"net/http"
	"time"

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

func (h *Handler) Dashboard(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	stats, err := h.svc.GetDashboard(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get dashboard", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to load dashboard")
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

func (h *Handler) Stats(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	stats, err := h.svc.GetStats(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to get stats", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to load stats")
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

func (h *Handler) RecentPosts(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	posts, err := h.svc.listPostsByStatus(ctx.Request.Context(), siteID, "", 10)
	if err != nil {
		h.log.Error("failed to list recent posts", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list posts")
		return
	}

	ctx.JSON(http.StatusOK, posts)
}

func (h *Handler) DraftPosts(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	posts, err := h.svc.listPostsByStatus(ctx.Request.Context(), siteID, "draft", 10)
	if err != nil {
		h.log.Error("failed to list draft posts", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list posts")
		return
	}

	ctx.JSON(http.StatusOK, posts)
}

func (h *Handler) ScheduledPosts(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	posts, err := h.svc.listPostsByStatus(ctx.Request.Context(), siteID, "scheduled", 10)
	if err != nil {
		h.log.Error("failed to list scheduled posts", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list posts")
		return
	}

	ctx.JSON(http.StatusOK, posts)
}

func (h *Handler) CreateTask(ctx *rest.Context) {
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

	var req CreateTaskRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}

	task, err := h.svc.CreateTask(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		h.log.Error("failed to create task", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create task")
		return
	}

	ctx.JSON(http.StatusCreated, task)
}

func (h *Handler) GetTask(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	task, err := h.svc.GetTask(ctx.Request.Context(), siteID, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "task not found")
		} else {
			h.log.Error("failed to get task", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get task")
		}
		return
	}

	ctx.JSON(http.StatusOK, task)
}

func (h *Handler) ListTasks(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := TaskStatus(ctx.Request.URL.Query().Get("status"))

	tasks, err := h.svc.ListTasks(ctx.Request.Context(), siteID, status)
	if err != nil {
		h.log.Error("failed to list tasks", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list tasks")
		return
	}

	ctx.JSON(http.StatusOK, tasks)
}

func (h *Handler) UpdateTask(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	var req UpdateTaskRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	task, err := h.svc.UpdateTask(ctx.Request.Context(), siteID, taskID, req)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "task not found")
		} else {
			h.log.Error("failed to update task", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update task")
		}
		return
	}

	ctx.JSON(http.StatusOK, task)
}

func (h *Handler) DeleteTask(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	if err := h.svc.DeleteTask(ctx.Request.Context(), siteID, taskID); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "task not found")
		} else {
			h.log.Error("failed to delete task", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete task")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SaveRevision(ctx *rest.Context) {
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

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	var req CreateRevisionRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	revision, err := h.svc.SaveRevision(ctx.Request.Context(), siteID, postID, userID, req)
	if err != nil {
		h.log.Error("failed to save revision", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to save revision")
		return
	}

	ctx.JSON(http.StatusCreated, revision)
}

func (h *Handler) ListRevisions(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	revisions, err := h.svc.ListRevisions(ctx.Request.Context(), siteID, postID)
	if err != nil {
		h.log.Error("failed to list revisions", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list revisions")
		return
	}

	ctx.JSON(http.StatusOK, revisions)
}

func (h *Handler) RestoreRevision(ctx *rest.Context) {
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

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	revID, err := uuid.Parse(chi.URLParam(ctx.Request, "revID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid revision ID")
		return
	}

	revision, err := h.svc.RestoreRevision(ctx.Request.Context(), siteID, postID, revID, userID)
	if err != nil {
		if errors.Is(err, ErrRevisionNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "revision not found")
		} else {
			h.log.Error("failed to restore revision", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to restore revision")
		}
		return
	}

	ctx.JSON(http.StatusOK, revision)
}

func (h *Handler) RequestApproval(ctx *rest.Context) {
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

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	req, err := h.svc.RequestApproval(ctx.Request.Context(), siteID, postID, userID)
	if err != nil {
		h.log.Error("failed to request approval", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to request approval")
		return
	}

	ctx.JSON(http.StatusCreated, req)
}

func (h *Handler) ReviewApproval(ctx *rest.Context) {
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

	postID, err := uuid.Parse(chi.URLParam(ctx.Request, "postID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid post ID")
		return
	}

	approvalID, err := uuid.Parse(chi.URLParam(ctx.Request, "approvalID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid approval ID")
		return
	}

	var req ApprovalActionRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Status != ApprovalStatusApproved && req.Status != ApprovalStatusRejected {
		ctx.Error(http.StatusBadRequest, "INVALID_STATUS", "status must be approved or rejected")
		return
	}

	result, err := h.svc.ReviewApproval(ctx.Request.Context(), siteID, postID, approvalID, userID, req)
	if err != nil {
		h.log.Error("failed to review approval", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to review approval")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) ListApprovals(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ApprovalStatus(ctx.Request.URL.Query().Get("status"))

	approvals, err := h.svc.ListApprovals(ctx.Request.Context(), siteID, status)
	if err != nil {
		h.log.Error("failed to list approvals", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list approvals")
		return
	}

	ctx.JSON(http.StatusOK, approvals)
}

func (h *Handler) CreateCalendarEvent(ctx *rest.Context) {
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

	var req CreateCalendarEventRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Title == "" || req.EventDate == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title and event_date are required")
		return
	}

	event, err := h.svc.CreateCalendarEvent(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		h.log.Error("failed to create calendar event", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create calendar event")
		return
	}

	ctx.JSON(http.StatusCreated, event)
}

func (h *Handler) ListCalendarEvents(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	startDate := ctx.Request.URL.Query().Get("start_date")
	endDate := ctx.Request.URL.Query().Get("end_date")

	events, err := h.svc.ListCalendarEvents(ctx.Request.Context(), siteID, startDate, endDate)
	if err != nil {
		h.log.Error("failed to list calendar events", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list calendar events")
		return
	}

	ctx.JSON(http.StatusOK, events)
}

func (h *Handler) UpdateCalendarEvent(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid calendar event ID")
		return
	}

	var req UpdateCalendarEventRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	event, err := h.svc.UpdateCalendarEvent(ctx.Request.Context(), siteID, eventID, req)
	if err != nil {
		if errors.Is(err, ErrCalendarEventNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "calendar event not found")
		} else {
			h.log.Error("failed to update calendar event", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update calendar event")
		}
		return
	}

	ctx.JSON(http.StatusOK, event)
}

func (h *Handler) DeleteCalendarEvent(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid calendar event ID")
		return
	}

	if err := h.svc.DeleteCalendarEvent(ctx.Request.Context(), siteID, eventID); err != nil {
		h.log.Error("failed to delete calendar event", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete calendar event")
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListWidgets(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	widgets, err := h.svc.ListWidgets(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list widgets", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list widgets")
		return
	}

	ctx.JSON(http.StatusOK, widgets)
}

func (h *Handler) UpdateWidget(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	widgetID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid widget ID")
		return
	}

	var req UpdateWidgetRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	widget, err := h.svc.UpdateWidget(ctx.Request.Context(), siteID, widgetID, req)
	if err != nil {
		if errors.Is(err, ErrWidgetNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "widget not found")
		} else {
			h.log.Error("failed to update widget", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update widget")
		}
		return
	}

	ctx.JSON(http.StatusOK, widget)
}

func (h *Handler) GetTasksForDate(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	dateStr := chi.URLParam(ctx.Request, "date")
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_DATE", "invalid date format, use YYYY-MM-DD")
		return
	}

	tasks, err := h.svc.ListTasks(ctx.Request.Context(), siteID, "")
	if err != nil {
		h.log.Error("failed to get tasks for date", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get tasks")
		return
	}

	var filtered []Task
	for _, t := range tasks {
		if t.DueDate != nil {
			d := t.DueDate.Format("2006-01-02")
			if d == dateStr {
				filtered = append(filtered, t)
			}
		}
	}
	if filtered == nil {
		filtered = []Task{}
	}

	ctx.JSON(http.StatusOK, filtered)
}

func (h *Handler) AIInsights(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	h.log.Info("AI insights endpoint called (placeholder)", "site_id", siteID)

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":  "AI integration endpoint ready",
		"features": []string{"content_suggestions", "seo_analysis", "auto_tagging", "summary_generation", "translation"},
		"status":   "coming_soon",
	})
}
