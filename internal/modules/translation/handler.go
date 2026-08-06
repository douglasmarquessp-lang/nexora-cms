package translation

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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

	job, err := h.svc.CreateJob(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrTitleRequired):
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "title is required")
		case errors.Is(err, ErrTargetSiteRequired):
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "target_site_id is required")
		case errors.Is(err, ErrInvalidLanguage):
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		default:
			h.log.Error("failed to create translation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create translation job")
		}
		return
	}

	ctx.JSON(http.StatusCreated, job)
}

func (h *Handler) ListJobs(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	jobs, err := h.svc.ListJobs(ctx.Request.Context(), siteID, status, limit, offset)
	if err != nil {
		h.log.Error("failed to list translation jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list translation jobs")
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"jobs": jobs, "total": len(jobs)})
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

	job, err := h.svc.GetJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		} else {
			h.log.Error("failed to get translation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get translation job")
		}
		return
	}
	ctx.JSON(http.StatusOK, job)
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
		switch {
		case errors.Is(err, ErrJobNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		case errors.Is(err, ErrInvalidStatus):
			ctx.Error(http.StatusConflict, "INVALID_STATUS", "job is already running")
		default:
			h.log.Error("failed to start translation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to start translation job")
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

	job, err := h.svc.CancelJob(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		case errors.Is(err, ErrInvalidStatus):
			ctx.Error(http.StatusConflict, "INVALID_STATUS", "job cannot be cancelled in its current state")
		default:
			h.log.Error("failed to cancel translation job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to cancel translation job")
		}
		return
	}
	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) GetStages(ctx *rest.Context) {
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

	stages, err := h.svc.GetStages(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		} else {
			h.log.Error("failed to list translation stages", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list translation stages")
		}
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"stages": stages})
}

func (h *Handler) GetScore(ctx *rest.Context) {
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

	score, err := h.svc.GetScore(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		} else {
			h.log.Error("failed to get translation score", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get translation score")
		}
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"score": score})
}

// --- Stage decisions (approve / reject the current stage) ---

type rejectRequest struct {
	Feedback string `json:"feedback"`
}

func (h *Handler) ApproveStage(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}
	stageID, err := uuid.Parse(chi.URLParam(ctx.Request, "stageID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid stage ID")
		return
	}

	job, err := h.svc.ApproveStage(ctx.Request.Context(), siteID, stageID)
	if err != nil {
		switch {
		case errors.Is(err, ErrStageNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation stage not found")
		case errors.Is(err, ErrJobNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		case errors.Is(err, ErrInvalidStatus):
			ctx.Error(http.StatusConflict, "INVALID_STATUS", "job is not waiting for review")
		default:
			h.log.Error("failed to approve translation stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to approve translation stage")
		}
		return
	}
	ctx.JSON(http.StatusOK, job)
}

func (h *Handler) RejectStage(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}
	stageID, err := uuid.Parse(chi.URLParam(ctx.Request, "stageID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid stage ID")
		return
	}

	var req rejectRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	job, err := h.svc.RejectStage(ctx.Request.Context(), siteID, stageID, req.Feedback)
	if err != nil {
		switch {
		case errors.Is(err, ErrStageNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation stage not found")
		case errors.Is(err, ErrJobNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation job not found")
		case errors.Is(err, ErrInvalidStatus):
			ctx.Error(http.StatusConflict, "INVALID_STATUS", "job is not waiting for review")
		default:
			h.log.Error("failed to reject translation stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to reject translation stage")
		}
		return
	}
	ctx.JSON(http.StatusOK, job)
}

// --- Language detection (deterministic) ---

type detectRequest struct {
	Text string `json:"text"`
}

func (h *Handler) DetectLanguage(ctx *rest.Context) {
	var req detectRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "text is required")
		return
	}
	lang, confidence := DetectLanguage(req.Text)
	ctx.JSON(http.StatusOK, DetectLanguageResult{Language: lang, Confidence: confidence})
}

// --- Glossary ---

func (h *Handler) CreateGlossaryTerm(ctx *rest.Context) {
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

	var req CreateGlossaryTermRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	term, err := h.svc.CreateGlossaryTerm(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidGlossary):
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "source and target terms are required")
		case errors.Is(err, ErrInvalidLanguage):
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		case errors.Is(err, ErrGlossaryDuplicate):
			ctx.Error(http.StatusConflict, "DUPLICATE", "glossary term already exists for this scope and direction")
		default:
			h.log.Error("failed to create glossary term", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create glossary term")
		}
		return
	}
	ctx.JSON(http.StatusCreated, term)
}

func (h *Handler) ListGlossaryTerms(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var projectID *uuid.UUID
	if v := ctx.Request.URL.Query().Get("project_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
			return
		}
		projectID = &id
	}

	terms, err := h.svc.ListGlossaryTerms(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to list glossary terms", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list glossary terms")
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"terms": terms, "total": len(terms)})
}

func (h *Handler) UpdateGlossaryTerm(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}
	termID, err := uuid.Parse(chi.URLParam(ctx.Request, "termID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid term ID")
		return
	}

	var req UpdateGlossaryTermRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	term, err := h.svc.UpdateGlossaryTerm(ctx.Request.Context(), siteID, termID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrGlossaryNotFound):
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "glossary term not found")
		case errors.Is(err, ErrInvalidGlossary):
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "source and target terms are required")
		default:
			h.log.Error("failed to update glossary term", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update glossary term")
		}
		return
	}
	ctx.JSON(http.StatusOK, term)
}

func (h *Handler) DeleteGlossaryTerm(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}
	termID, err := uuid.Parse(chi.URLParam(ctx.Request, "termID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid term ID")
		return
	}

	if err := h.svc.DeleteGlossaryTerm(ctx.Request.Context(), siteID, termID); err != nil {
		if errors.Is(err, ErrGlossaryNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "glossary term not found")
		} else {
			h.log.Error("failed to delete glossary term", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete glossary term")
		}
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
}
