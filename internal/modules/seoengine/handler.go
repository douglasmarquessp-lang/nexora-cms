package seoengine

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

// --- Projects ---

func (h *Handler) CreateProject(ctx *rest.Context) {
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

	var req CreateProjectRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Title == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "project title is required")
		return
	}

	project, err := h.svc.CreateProject(ctx.Request.Context(), siteID, userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else {
			h.log.Error("failed to create seo project", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create seo project")
		}
		return
	}

	ctx.JSON(http.StatusCreated, project)
}

func (h *Handler) GetProject(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	project, err := h.svc.GetProject(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo project not found")
		} else {
			h.log.Error("failed to get seo project", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get seo project")
		}
		return
	}

	ctx.JSON(http.StatusOK, project)
}

func (h *Handler) ListProjects(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	status := ctx.Request.URL.Query().Get("status")
	language := ctx.Request.URL.Query().Get("language")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	projects, err := h.svc.ListProjects(ctx.Request.Context(), siteID, status, language, limit, offset)
	if err != nil {
		h.log.Error("failed to list seo projects", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list seo projects")
		return
	}

	ctx.JSON(http.StatusOK, projects)
}

func updateEntity[Req, Res any](h *Handler, ctx *rest.Context, urlParam, invalidIDMsg string,
	svcMethod func(context.Context, uuid.UUID, uuid.UUID, Req) (Res, error),
	notFoundErr error, notFoundMsg, logMsg string) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	entityID, err := uuid.Parse(chi.URLParam(ctx.Request, urlParam))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", invalidIDMsg)
		return
	}

	var req Req
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	result, err := svcMethod(ctx.Request.Context(), siteID, entityID, req)
	if err != nil {
		if errors.Is(err, notFoundErr) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", notFoundMsg)
		} else {
			h.log.Error(logMsg, "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", logMsg)
		}
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) UpdateProject(ctx *rest.Context) {
	updateEntity(h, ctx, "id", "invalid project ID",
		func(c context.Context, s, id uuid.UUID, req UpdateProjectRequest) (*SEOProject, error) { return h.svc.UpdateProject(c, s, id, req) },
		ErrProjectNotFound, "seo project not found", "failed to update seo project")
}

func (h *Handler) UpdateImprovement(ctx *rest.Context) {
	updateEntity(h, ctx, "improvementID", "invalid improvement ID",
		func(c context.Context, s, id uuid.UUID, req UpdateImprovementRequest) (*SEOImprovement, error) { return h.svc.UpdateImprovement(c, s, id, req) },
		ErrImprovementNotFound, "improvement not found", "failed to update improvement")
}

func (h *Handler) DeleteProject(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	err = h.svc.DeleteProject(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo project not found")
		} else {
			h.log.Error("failed to delete seo project", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete seo project")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// --- Audit ---

func (h *Handler) RunFullAudit(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	audit, err := h.svc.RunFullAudit(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo project not found")
		} else {
			h.log.Error("failed to run audit", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to run audit")
		}
		return
	}

	ctx.JSON(http.StatusOK, audit)
}

func (h *Handler) GetAudit(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	auditID, err := uuid.Parse(chi.URLParam(ctx.Request, "auditID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid audit ID")
		return
	}

	audit, err := h.svc.GetAudit(ctx.Request.Context(), siteID, auditID)
	if err != nil {
		if errors.Is(err, ErrAuditNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo audit not found")
		} else {
			h.log.Error("failed to get audit", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get audit")
		}
		return
	}

	ctx.JSON(http.StatusOK, audit)
}

func (h *Handler) GetProjectAudits(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	audits, err := h.svc.GetProjectAudits(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to list audits", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list audits")
		return
	}

	ctx.JSON(http.StatusOK, audits)
}

// --- Scores ---

func (h *Handler) GetScores(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	scores, err := h.svc.GetScores(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		if errors.Is(err, ErrScoreNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo scores not found")
		} else {
			h.log.Error("failed to get scores", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get scores")
		}
		return
	}

	ctx.JSON(http.StatusOK, scores)
}

// --- Analysis ---

func (h *Handler) AnalyzeKeywords(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req KeywordAnalysisRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	result, err := h.svc.AnalyzeKeywords(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to analyze keywords", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to analyze keywords")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) AnalyzeContent(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	result, err := h.svc.AnalyzeContent(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to analyze content", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to analyze content")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) AnalyzeTechnical(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	result, err := h.svc.AnalyzeTechnical(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to analyze technical seo", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to analyze technical seo")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// --- Clusters ---

func (h *Handler) CreateCluster(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreateClusterRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "cluster name is required")
		return
	}

	cluster, err := h.svc.CreateCluster(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to create cluster", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create cluster")
		return
	}

	ctx.JSON(http.StatusCreated, cluster)
}

func (h *Handler) ListClusters(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	clusters, err := h.svc.ListClusters(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list clusters", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list clusters")
		return
	}

	ctx.JSON(http.StatusOK, clusters)
}

// --- Improvements ---

func (h *Handler) AddImprovement(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	var req AddImprovementRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Issue == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "issue is required")
		return
	}
	if req.Suggestion == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "suggestion is required")
		return
	}

	improvement, err := h.svc.AddImprovement(ctx.Request.Context(), siteID, projectID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCategory) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid improvement category")
		} else {
			h.log.Error("failed to add improvement", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to add improvement")
		}
		return
	}

	ctx.JSON(http.StatusCreated, improvement)
}

func (h *Handler) ListImprovements(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	category := ctx.Request.URL.Query().Get("category")
	status := ctx.Request.URL.Query().Get("status")

	improvements, err := h.svc.ListImprovements(ctx.Request.Context(), siteID, projectID, category, status)
	if err != nil {
		h.log.Error("failed to list improvements", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list improvements")
		return
	}

	ctx.JSON(http.StatusOK, improvements)
}

// --- Suggestions ---

func (h *Handler) GetLinkingSuggestions(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	suggestions, err := h.svc.GetInternalLinkingSuggestions(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to get linking suggestions", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get linking suggestions")
		return
	}

	ctx.JSON(http.StatusOK, suggestions)
}

func (h *Handler) GetSchemaRecommendations(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	recommendations, err := h.svc.GetSchemaRecommendations(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to get schema recommendations", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get schema recommendations")
		return
	}

	ctx.JSON(http.StatusOK, recommendations)
}

func (h *Handler) GenerateChecklist(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	checklist, err := h.svc.GenerateChecklist(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to generate checklist", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to generate checklist")
		return
	}

	ctx.JSON(http.StatusOK, checklist)
}

// --- Detection ---

func (h *Handler) DetectOrphans(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	orphans, err := h.svc.DetectOrphanArticles(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to detect orphans", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to detect orphans")
		return
	}

	ctx.JSON(http.StatusOK, orphans)
}

func (h *Handler) DetectDuplicates(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid project ID")
		return
	}

	duplicates, err := h.svc.DetectDuplicates(ctx.Request.Context(), siteID, projectID)
	if err != nil {
		h.log.Error("failed to detect duplicates", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to detect duplicates")
		return
	}

	ctx.JSON(http.StatusOK, duplicates)
}

func (h *Handler) DetectCannibalization(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	issues, err := h.svc.DetectCannibalization(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to detect cannibalization", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to detect cannibalization")
		return
	}

	ctx.JSON(http.StatusOK, issues)
}

func (h *Handler) DetectContentGaps(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	gaps, err := h.svc.DetectContentGaps(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to detect content gaps", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to detect content gaps")
		return
	}

	ctx.JSON(http.StatusOK, gaps)
}

// --- Dashboard ---

func (h *Handler) GetDashboardStats(ctx *rest.Context) {
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
