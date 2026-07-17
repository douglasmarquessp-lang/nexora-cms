package writer

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

	var req CreateArticleJobRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Headline == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "headline is required")
		return
	}
	if req.Language != "pt" && req.Language != "en" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		return
	}

	job, err := h.svc.CreateJob(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrStyleNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "writing style not found")
		} else {
			h.log.Error("failed to create article job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create article job")
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
		if errors.Is(err, ErrWritingJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "article job not found")
		} else {
			h.log.Error("failed to get article job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get article job")
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
		h.log.Error("failed to list article jobs", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list article jobs")
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

	var req UpdateArticleJobRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	job, err := h.svc.UpdateJob(ctx.Request.Context(), siteID, jobID, req)
	if err != nil {
		if errors.Is(err, ErrWritingJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "article job not found")
		} else {
			h.log.Error("failed to update article job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update article job")
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
		if errors.Is(err, ErrWritingJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "article job not found")
		} else {
			h.log.Error("failed to delete article job", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete article job")
		}
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) CreateOutline(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	var req CreateOutlineRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	outlines, err := h.svc.CreateOutline(ctx.Request.Context(), jobID, req)
	if err != nil {
		h.log.Error("failed to create outline", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create outline")
		return
	}

	ctx.JSON(http.StatusCreated, outlines)
}

func (h *Handler) ListOutline(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	outlines, err := h.svc.ListOutline(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to list outline", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list outline")
		return
	}

	ctx.JSON(http.StatusOK, outlines)
}

func (h *Handler) CreateSection(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	var req CreateSectionRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	section, err := h.svc.CreateSection(ctx.Request.Context(), jobID, req)
	if err != nil {
		h.log.Error("failed to create section", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create section")
		return
	}

	ctx.JSON(http.StatusCreated, section)
}

func (h *Handler) ListSections(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	sections, err := h.svc.ListSections(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to list sections", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list sections")
		return
	}

	ctx.JSON(http.StatusOK, sections)
}

func (h *Handler) GetSection(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	sectionID, err := uuid.Parse(chi.URLParam(ctx.Request, "sectionID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid section ID")
		return
	}

	section, err := h.svc.GetSection(ctx.Request.Context(), jobID, sectionID)
	if err != nil {
		if errors.Is(err, ErrSectionNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "section not found")
		} else {
			h.log.Error("failed to get section", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get section")
		}
		return
	}

	ctx.JSON(http.StatusOK, section)
}

func (h *Handler) UpdateSection(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	sectionID, err := uuid.Parse(chi.URLParam(ctx.Request, "sectionID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid section ID")
		return
	}

	var req UpdateSectionRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	section, err := h.svc.UpdateSection(ctx.Request.Context(), jobID, sectionID, req)
	if err != nil {
		if errors.Is(err, ErrSectionNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "section not found")
		} else {
			h.log.Error("failed to update section", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update section")
		}
		return
	}

	ctx.JSON(http.StatusOK, section)
}

func (h *Handler) CreateVersion(ctx *rest.Context) {
	_, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	var req struct {
		ChangeLog string `json:"change_log"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	version, err := h.svc.CreateVersion(ctx.Request.Context(), jobID, userID, req.ChangeLog)
	if err != nil {
		if errors.Is(err, ErrWritingJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "article job not found")
		} else {
			h.log.Error("failed to create version", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create version")
		}
		return
	}

	ctx.JSON(http.StatusCreated, version)
}

func (h *Handler) ListVersions(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	versions, err := h.svc.ListVersions(ctx.Request.Context(), jobID)
	if err != nil {
		h.log.Error("failed to list versions", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list versions")
		return
	}

	ctx.JSON(http.StatusOK, versions)
}

func (h *Handler) GetVersion(ctx *rest.Context) {
	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	versionID, err := uuid.Parse(chi.URLParam(ctx.Request, "versionID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid version ID")
		return
	}

	version, err := h.svc.GetVersion(ctx.Request.Context(), jobID, versionID)
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "version not found")
		} else {
			h.log.Error("failed to get version", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get version")
		}
		return
	}

	ctx.JSON(http.StatusOK, version)
}

func (h *Handler) RestoreVersion(ctx *rest.Context) {
	_, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	userID, ok := middleware.GetUserID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid job ID")
		return
	}

	versionID, err := uuid.Parse(chi.URLParam(ctx.Request, "versionID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid version ID")
		return
	}

	version, err := h.svc.RestoreVersion(ctx.Request.Context(), jobID, versionID, userID)
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "version not found")
		} else {
			h.log.Error("failed to restore version", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to restore version")
		}
		return
	}

	ctx.JSON(http.StatusOK, version)
}

func (h *Handler) ListStyles(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	styles, err := h.svc.ListStyles(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list styles", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list styles")
		return
	}

	ctx.JSON(http.StatusOK, styles)
}
