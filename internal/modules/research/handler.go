package research

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/ai"
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

// DeepResearchRequest is the input of POST /research/deep.
type DeepResearchRequest struct {
	Topic    string `json:"topic"`
	Language string `json:"language"`
}

// DeepResearch runs the full research workflow (cache → search → ranking →
// fact base → briefing) and returns the report.
func (h *Handler) DeepResearch(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req DeepResearchRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	report, err := h.svc.DeepResearch(ctx.Request.Context(), siteID, strings.TrimSpace(req.Topic), req.Language)
	if err != nil {
		if errors.Is(err, ErrTopicRequired) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "topic is required")
		} else if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else {
			h.log.Error("deep research failed", "topic", req.Topic, "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "deep research failed")
		}
		return
	}

	status := http.StatusOK
	if !report.Cached {
		status = http.StatusCreated
	}
	ctx.JSON(status, report)
}

// GetDeepReport returns the persisted report of a research job.
func (h *Handler) GetDeepReport(ctx *rest.Context) {
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

	detail, err := h.svc.GetJobDetail(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrResearchJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "research job not found")
		} else {
			h.log.Error("failed to get deep research report", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get deep research report")
		}
		return
	}

	report := DeepResearchReport{
		ResearchJob: detail.ResearchJob,
		Sources:     detail.Sources,
	}
	if detail.Briefing != nil {
		if doc, ok := briefingDocFromMap(detail.Briefing.StructuredBriefing); ok {
			report.Briefing = &doc
		}
	}
	report.Facts = h.loadFactBase(ctx, jobID)

	ctx.JSON(http.StatusOK, report)
}

// GetFactBase returns the persisted fact base of a research job.
func (h *Handler) GetFactBase(ctx *rest.Context) {
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

	facts, err := h.svc.GetFactBase(ctx.Request.Context(), siteID, jobID)
	if err != nil {
		if errors.Is(err, ErrResearchJobNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "research job not found")
		} else {
			h.log.Error("failed to get fact base", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get fact base")
		}
		return
	}

	ctx.JSON(http.StatusOK, facts)
}

// GetCachedResearch returns the cached research for a topic (24h TTL).
func (h *Handler) GetCachedResearch(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	topic := chi.URLParam(ctx.Request, "topic")
	if topic == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "topic is required")
		return
	}
	language := ctx.Request.URL.Query().Get("language")
	if language == "" {
		language = "pt"
	}

	cached, err := h.svc.GetCachedResearch(ctx.Request.Context(), siteID, topic, language)
	if err != nil {
		if errors.Is(err, ErrCacheEntryNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "no cached research for topic")
		} else {
			h.log.Error("failed to get cached research", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get cached research")
		}
		return
	}

	ctx.JSON(http.StatusOK, cached)
}

// ListReliability returns the built-in domain reliability ranking.
func (h *Handler) ListReliability(ctx *rest.Context) {
	ctx.JSON(http.StatusOK, ai.DefaultReliabilityScores())
}

func (h *Handler) loadFactBase(ctx *rest.Context, jobID uuid.UUID) []FactBaseEntry {
	facts, err := h.svc.GetFactBase(ctx.Request.Context(), uuid.Nil, jobID)
	if err != nil {
		return nil
	}
	return facts
}

func briefingDocFromMap(m map[string]interface{}) (ResearchBriefingDoc, bool) {
	if len(m) == 0 {
		return ResearchBriefingDoc{}, false
	}
	var doc ResearchBriefingDoc
	doc.Topic, _ = m["topic"].(string)
	doc.Summary, _ = m["summary"].(string)
	doc.KeyPoints = stringSlice(m["key_points"])
	doc.DataFound = stringSlice(m["data_found"])
	doc.Statistics = stringSlice(m["statistics"])
	doc.Dates = stringSlice(m["dates"])
	doc.Companies = stringSlice(m["companies"])
	doc.Products = stringSlice(m["products"])
	doc.Conclusions = stringSlice(m["conclusions"])
	return doc, true
}

func stringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if arr, ok := v.([]interface{}); ok {
		var out []string
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
