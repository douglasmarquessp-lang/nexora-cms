package editorialengine

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

// --- Pipeline ---

func (h *Handler) CreatePipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreatePipelineRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	pipeline, err := h.svc.CreatePipeline(ctx.Request.Context(), siteID, req)
	if err != nil {
		if errors.Is(err, ErrJobAlreadyInPipeline) {
			ctx.Error(http.StatusConflict, "CONFLICT", "article job already has a pipeline")
		} else {
			h.log.Error("failed to create pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create pipeline")
		}
		return
	}

	ctx.JSON(http.StatusCreated, pipeline)
}

func (h *Handler) GetPipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pipelineID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline ID")
		return
	}

	pipeline, err := h.svc.GetPipeline(ctx.Request.Context(), siteID, pipelineID)
	if err != nil {
		if errors.Is(err, ErrPipelineNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline not found")
		} else {
			h.log.Error("failed to get pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get pipeline")
		}
		return
	}

	ctx.JSON(http.StatusOK, pipeline)
}

func (h *Handler) ListPipelines(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	stage := ctx.Request.URL.Query().Get("stage")
	status := ctx.Request.URL.Query().Get("status")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	pipelines, err := h.svc.ListPipelines(ctx.Request.Context(), siteID, stage, status, limit, offset)
	if err != nil {
		h.log.Error("failed to list pipelines", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list pipelines")
		return
	}

	ctx.JSON(http.StatusOK, pipelines)
}

func (h *Handler) UpdatePipeline(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pipelineID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline ID")
		return
	}

	var req UpdatePipelineRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	detail, err := h.svc.UpdatePipeline(ctx.Request.Context(), siteID, pipelineID, req)
	if err != nil {
		if errors.Is(err, ErrPipelineNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "pipeline not found")
		} else if errors.Is(err, ErrInvalidStage) {
			ctx.Error(http.StatusBadRequest, "INVALID_STAGE", "invalid pipeline stage")
		} else {
			h.log.Error("failed to update pipeline", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update pipeline")
		}
		return
	}

	ctx.JSON(http.StatusOK, detail)
}

func (h *Handler) ListPipelineStages(ctx *rest.Context) {
	_, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pipelineID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline ID")
		return
	}

	stages, err := h.svc.ListPipelineStages(ctx.Request.Context(), pipelineID)
	if err != nil {
		h.log.Error("failed to list stages", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list stages")
		return
	}

	ctx.JSON(http.StatusOK, stages)
}

func (h *Handler) GetPipelineStage(ctx *rest.Context) {
	_, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pipelineID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline ID")
		return
	}

	stageID, err := uuid.Parse(chi.URLParam(ctx.Request, "stageID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid stage ID")
		return
	}

	stage, err := h.svc.GetPipelineStage(ctx.Request.Context(), pipelineID, stageID)
	if err != nil {
		if errors.Is(err, ErrStageNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "stage not found")
		} else {
			h.log.Error("failed to get stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get stage")
		}
		return
	}

	ctx.JSON(http.StatusOK, stage)
}

func (h *Handler) UpdatePipelineStage(ctx *rest.Context) {
	_, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	pipelineID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid pipeline ID")
		return
	}

	stageID, err := uuid.Parse(chi.URLParam(ctx.Request, "stageID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid stage ID")
		return
	}

	var req UpdateStageRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	stage, err := h.svc.UpdatePipelineStage(ctx.Request.Context(), pipelineID, stageID, req)
	if err != nil {
		if errors.Is(err, ErrStageNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "stage not found")
		} else {
			h.log.Error("failed to update stage", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update stage")
		}
		return
	}

	ctx.JSON(http.StatusOK, stage)
}

// --- Style Rules ---

func (h *Handler) GetStyleRules(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	rules, err := h.svc.GetStyleRules(ctx.Request.Context(), siteID)
	if err != nil {
		if errors.Is(err, ErrStyleRulesNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "style rules not found")
		} else {
			h.log.Error("failed to get style rules", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get style rules")
		}
		return
	}

	ctx.JSON(http.StatusOK, rules)
}

func (h *Handler) UpsertStyleRules(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req UpdateStyleRulesRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	rules, err := h.svc.UpsertStyleRules(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to upsert style rules", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to upsert style rules")
		return
	}

	ctx.JSON(http.StatusOK, rules)
}

// --- SEO ---

func (h *Handler) CreateSEOData(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreateSEODataRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	seo, err := h.svc.CreateSEOData(ctx.Request.Context(), siteID, articleJobID, req)
	if err != nil {
		h.log.Error("failed to create seo data", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create seo data")
		return
	}

	ctx.JSON(http.StatusCreated, seo)
}

func (h *Handler) GetSEOData(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	seo, err := h.svc.GetSEOData(ctx.Request.Context(), siteID, articleJobID)
	if err != nil {
		if errors.Is(err, ErrSEONotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo data not found")
		} else {
			h.log.Error("failed to get seo data", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get seo data")
		}
		return
	}

	ctx.JSON(http.StatusOK, seo)
}

func (h *Handler) UpdateSEOData(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	var req UpdateSEODataRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	seo, err := h.svc.UpdateSEOData(ctx.Request.Context(), siteID, articleJobID, req)
	if err != nil {
		if errors.Is(err, ErrSEONotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "seo data not found")
		} else {
			h.log.Error("failed to update seo data", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update seo data")
		}
		return
	}

	ctx.JSON(http.StatusOK, seo)
}

// --- Quality ---

func (h *Handler) CreateQualityScore(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreateQualityScoreRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	score, err := h.svc.CreateQualityScore(ctx.Request.Context(), siteID, articleJobID, req)
	if err != nil {
		h.log.Error("failed to create quality score", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create quality score")
		return
	}

	ctx.JSON(http.StatusCreated, score)
}

func (h *Handler) GetQualityScore(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	score, err := h.svc.GetQualityScore(ctx.Request.Context(), siteID, articleJobID)
	if err != nil {
		if errors.Is(err, ErrQualityNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "quality score not found")
		} else {
			h.log.Error("failed to get quality score", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get quality score")
		}
		return
	}

	ctx.JSON(http.StatusOK, score)
}

func (h *Handler) ListQualityScores(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	scores, err := h.svc.ListQualityScores(ctx.Request.Context(), siteID, articleJobID)
	if err != nil {
		h.log.Error("failed to list quality scores", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list quality scores")
		return
	}

	ctx.JSON(http.StatusOK, scores)
}

// --- Translation ---

func (h *Handler) CreateTranslation(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req CreateTranslationRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	translation, err := h.svc.CreateTranslation(ctx.Request.Context(), siteID, articleJobID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidTranslationDir) {
			ctx.Error(http.StatusBadRequest, "INVALID_DIRECTION", "translation must be PT↔EN")
		} else {
			h.log.Error("failed to create translation", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create translation")
		}
		return
	}

	ctx.JSON(http.StatusCreated, translation)
}

func (h *Handler) GetTranslation(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	translationID, err := uuid.Parse(chi.URLParam(ctx.Request, "translationID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid translation ID")
		return
	}

	translation, err := h.svc.GetTranslation(ctx.Request.Context(), siteID, translationID)
	if err != nil {
		if errors.Is(err, ErrTranslationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation not found")
		} else {
			h.log.Error("failed to get translation", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get translation")
		}
		return
	}

	ctx.JSON(http.StatusOK, translation)
}

func (h *Handler) ListTranslations(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	translations, err := h.svc.ListTranslations(ctx.Request.Context(), siteID, articleJobID)
	if err != nil {
		h.log.Error("failed to list translations", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list translations")
		return
	}

	ctx.JSON(http.StatusOK, translations)
}

func (h *Handler) UpdateTranslation(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	translationID, err := uuid.Parse(chi.URLParam(ctx.Request, "translationID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid translation ID")
		return
	}

	var req UpdateTranslationRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	translation, err := h.svc.UpdateTranslation(ctx.Request.Context(), siteID, translationID, req)
	if err != nil {
		if errors.Is(err, ErrTranslationNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "translation not found")
		} else {
			h.log.Error("failed to update translation", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update translation")
		}
		return
	}

	ctx.JSON(http.StatusOK, translation)
}

// --- Prompt Data ---

func (h *Handler) GetPromptData(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	pd, err := h.svc.GetPromptData(ctx.Request.Context(), siteID, articleJobID)
	if err != nil {
		if errors.Is(err, ErrPromptDataNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "prompt data not found")
		} else {
			h.log.Error("failed to get prompt data", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get prompt data")
		}
		return
	}

	ctx.JSON(http.StatusOK, pd)
}

func (h *Handler) CreatePromptData(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	articleJobID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid article job ID")
		return
	}

	pd, err := h.svc.CreatePromptData(ctx.Request.Context(), siteID, articleJobID)
	if err != nil {
		h.log.Error("failed to create prompt data", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create prompt data")
		return
	}

	ctx.JSON(http.StatusCreated, pd)
}
