package humanwriter

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

func (h *Handler) CreateProfile(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req ProfileCreateRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Slug == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "slug is required")
		return
	}
	if req.Name == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "name is required")
		return
	}

	profile, err := h.svc.CreateProfile(ctx.Request.Context(), siteID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidSlug) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "invalid slug")
		} else if errors.Is(err, ErrProfileExists) {
			ctx.Error(http.StatusConflict, "CONFLICT", "profile slug already exists")
		} else if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else {
			h.log.Error("failed to create profile", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create profile")
		}
		return
	}

	ctx.JSON(http.StatusCreated, profile)
}

func (h *Handler) GetProfile(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	profileID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid profile ID")
		return
	}

	profile, err := h.svc.GetProfile(ctx.Request.Context(), siteID, profileID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "profile not found")
		} else {
			h.log.Error("failed to get profile", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get profile")
		}
		return
	}

	ctx.JSON(http.StatusOK, profile)
}

func (h *Handler) ListProfiles(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	language := ctx.Request.URL.Query().Get("language")

	profiles, err := h.svc.ListProfiles(ctx.Request.Context(), siteID, language)
	if err != nil {
		h.log.Error("failed to list profiles", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list profiles")
		return
	}

	ctx.JSON(http.StatusOK, profiles)
}

func (h *Handler) UpdateProfile(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	profileID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid profile ID")
		return
	}

	var req ProfileUpdateRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	profile, err := h.svc.UpdateProfile(ctx.Request.Context(), siteID, profileID, req)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "profile not found")
		} else {
			h.log.Error("failed to update profile", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update profile")
		}
		return
	}

	ctx.JSON(http.StatusOK, profile)
}

func (h *Handler) DeleteProfile(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	profileID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid profile ID")
		return
	}

	err = h.svc.DeleteProfile(ctx.Request.Context(), siteID, profileID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "profile not found")
		} else {
			h.log.Error("failed to delete profile", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete profile")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListRules(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var profileID *uuid.UUID
	if pid := ctx.Request.URL.Query().Get("profile_id"); pid != "" {
		parsed, err := uuid.Parse(pid)
		if err == nil {
			profileID = &parsed
		}
	}

	rules, err := h.svc.ListRules(ctx.Request.Context(), siteID, profileID)
	if err != nil {
		h.log.Error("failed to list rules", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list rules")
		return
	}

	ctx.JSON(http.StatusOK, rules)
}

func (h *Handler) ToggleRule(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid rule ID")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := ctx.Decode(&body); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	rule, err := h.svc.ToggleRule(ctx.Request.Context(), siteID, ruleID, body.Enabled)
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "rule not found")
		} else {
			h.log.Error("failed to toggle rule", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to toggle rule")
		}
		return
	}

	ctx.JSON(http.StatusOK, rule)
}

func (h *Handler) CreatePersona(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req map[string]interface{}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	persona, err := h.svc.CreatePersona(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to create persona", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to create persona")
		return
	}

	ctx.JSON(http.StatusCreated, persona)
}

func (h *Handler) GetPersona(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	personaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid persona ID")
		return
	}

	persona, err := h.svc.GetPersona(ctx.Request.Context(), siteID, personaID)
	if err != nil {
		if errors.Is(err, ErrPersonaNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "persona not found")
		} else {
			h.log.Error("failed to get persona", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to get persona")
		}
		return
	}

	ctx.JSON(http.StatusOK, persona)
}

func (h *Handler) ListPersonas(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var profileID *uuid.UUID
	if pid := ctx.Request.URL.Query().Get("profile_id"); pid != "" {
		parsed, err := uuid.Parse(pid)
		if err == nil {
			profileID = &parsed
		}
	}
	language := ctx.Request.URL.Query().Get("language")

	personas, err := h.svc.ListPersonas(ctx.Request.Context(), siteID, profileID, language)
	if err != nil {
		h.log.Error("failed to list personas", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list personas")
		return
	}

	ctx.JSON(http.StatusOK, personas)
}

func (h *Handler) UpdatePersona(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	personaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid persona ID")
		return
	}

	var updates map[string]interface{}
	if err := ctx.Decode(&updates); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	persona, err := h.svc.UpdatePersona(ctx.Request.Context(), siteID, personaID, updates)
	if err != nil {
		if errors.Is(err, ErrPersonaNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "persona not found")
		} else {
			h.log.Error("failed to update persona", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to update persona")
		}
		return
	}

	ctx.JSON(http.StatusOK, persona)
}

func (h *Handler) DeletePersona(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	personaID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid persona ID")
		return
	}

	err = h.svc.DeletePersona(ctx.Request.Context(), siteID, personaID)
	if err != nil {
		if errors.Is(err, ErrPersonaNotFound) {
			ctx.Error(http.StatusNotFound, "NOT_FOUND", "persona not found")
		} else {
			h.log.Error("failed to delete persona", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to delete persona")
		}
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListVocabularySets(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	category := ctx.Request.URL.Query().Get("category")
	language := ctx.Request.URL.Query().Get("language")

	sets, err := h.svc.ListVocabularySets(ctx.Request.Context(), siteID, category, language)
	if err != nil {
		h.log.Error("failed to list vocabulary sets", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list vocabulary sets")
		return
	}

	ctx.JSON(http.StatusOK, sets)
}

func (h *Handler) ListTransitions(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	category := ctx.Request.URL.Query().Get("category")
	language := ctx.Request.URL.Query().Get("language")

	transitions, err := h.svc.ListTransitions(ctx.Request.Context(), siteID, category, language)
	if err != nil {
		h.log.Error("failed to list transitions", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list transitions")
		return
	}

	ctx.JSON(http.StatusOK, transitions)
}

func (h *Handler) ListPatterns(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	patternType := ctx.Request.URL.Query().Get("pattern_type")
	language := ctx.Request.URL.Query().Get("language")

	patterns, err := h.svc.ListPatterns(ctx.Request.Context(), siteID, patternType, language)
	if err != nil {
		h.log.Error("failed to list patterns", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list patterns")
		return
	}

	ctx.JSON(http.StatusOK, patterns)
}

func (h *Handler) ListTemplates(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	category := ctx.Request.URL.Query().Get("category")
	language := ctx.Request.URL.Query().Get("language")

	templates, err := h.svc.ListTemplates(ctx.Request.Context(), siteID, category, language)
	if err != nil {
		h.log.Error("failed to list templates", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list templates")
		return
	}

	ctx.JSON(http.StatusOK, templates)
}

func (h *Handler) ListHistory(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var profileID *uuid.UUID
	if pid := ctx.Request.URL.Query().Get("profile_id"); pid != "" {
		parsed, err := uuid.Parse(pid)
		if err == nil {
			profileID = &parsed
		}
	}
	language := ctx.Request.URL.Query().Get("language")
	limit, _ := strconv.Atoi(ctx.Request.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(ctx.Request.URL.Query().Get("offset"))

	records, err := h.svc.ListHistory(ctx.Request.Context(), siteID, profileID, language, limit, offset)
	if err != nil {
		h.log.Error("failed to list history", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list history")
		return
	}

	ctx.JSON(http.StatusOK, records)
}

func (h *Handler) Humanize(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req HumanizeRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Text == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "text is required")
		return
	}

	result, err := h.svc.Humanize(ctx.Request.Context(), siteID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidText) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "text is required")
		} else if errors.Is(err, ErrInvalidLanguage) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "language must be 'pt' or 'en'")
		} else {
			h.log.Error("failed to humanize text", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to humanize text")
		}
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) BatchHumanize(ctx *rest.Context) {
	siteID, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
		return
	}

	var req BatchHumanizeRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if len(req.Texts) == 0 {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "texts array is required")
		return
	}

	result, err := h.svc.BatchHumanize(ctx.Request.Context(), siteID, req)
	if err != nil {
		h.log.Error("failed to batch humanize", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to batch humanize")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (h *Handler) Analyze(ctx *rest.Context) {
	var req AnalyzeRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Text == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "text is required")
		return
	}

	result, err := h.svc.AnalyzeText(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidText) {
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "text is required")
		} else {
			h.log.Error("failed to analyze text", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to analyze text")
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
