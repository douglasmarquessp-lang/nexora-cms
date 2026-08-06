package freshness

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

func (h *Handler) siteID(ctx *rest.Context) (uuid.UUID, bool) {
	id, ok := middleware.GetSiteID(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusBadRequest, "MISSING_SITE", "site context required")
	}
	return id, ok
}

// Classify classifies the intent of a topic (NEWS/EVERGREEN/UPDATE/REVIEW/
// TUTORIAL) and returns the temporal research window.
func (h *Handler) Classify(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic    string `json:"topic"`
		Content  string `json:"content"`
		Language string `json:"language"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	ir, win, err := h.svc.ClassifyAndWindow(ctx.Request.Context(), siteID, req.Topic, req.Content, lang)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "CLASSIFY_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"intent":     ir.Intent,
		"confidence": ir.Confidence,
		"signals":    ir.Signals,
		"window":     win,
	})
}

// Score scores a list of sources for a given intent.
func (h *Handler) Score(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Intent        IntentType   `json:"intent"`
		TargetEntity  string       `json:"target_entity"`
		Entities      []EntityVersion `json:"entities"`
		ResearchJobID string       `json:"research_job_id"`
		Sources       []SourceInput `json:"sources"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Intent == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "intent is required")
		return
	}
	var jobID *uuid.UUID
	if req.ResearchJobID != "" {
		if id, err := uuid.Parse(req.ResearchJobID); err == nil {
			jobID = &id
		}
	}
	scored, err := h.svc.ScoreSources(ctx.Request.Context(), siteID, jobID, req.Intent, req.TargetEntity, req.Entities, req.Sources)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "SCORE_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"sources": scored})
}

// Obsolete flags outdated entity-version mentions in a text.
func (h *Handler) Obsolete(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Text     string          `json:"text"`
		Entities []EntityVersion `json:"entities"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	found := h.svc.DetectObsolete(req.Text, req.Entities)
	ctx.JSON(http.StatusOK, map[string]interface{}{"obsolete": found, "has_obsolete": len(found) > 0})
	_ = siteID
}

// SaveVersion stores a new article version.
func (h *Handler) SaveVersion(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req VersionRecord
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.PublicationID == uuid.Nil {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "publication_id is required")
		return
	}
	if req.Version == "" {
		req.Version = "v1"
	}
	if req.Intent == "" {
		req.Intent = IntentEvergreen
	}
	if err := h.svc.SaveVersion(ctx.Request.Context(), siteID, req); err != nil {
		if errors.Is(err, ErrDatabaseNotAvail) {
			ctx.Error(http.StatusServiceUnavailable, "DB_UNAVAILABLE", "database not available")
			return
		}
		h.log.Error("failed to save version", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to save version")
		return
	}
	ctx.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}

// ListVersions returns the version history of one publication.
func (h *Handler) ListVersions(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	pubID, err := uuid.Parse(chi.URLParam(ctx.Request, "publicationID"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid publication ID")
		return
	}
	versions, err := h.svc.ListVersions(ctx.Request.Context(), siteID, pubID)
	if err != nil {
		h.log.Error("failed to list versions", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list versions")
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"versions": versions})
}

// Dedup checks whether a topic was already covered the same day and registers
// the fingerprint.
func (h *Handler) Dedup(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic         string    `json:"topic"`
		Content       string    `json:"content"`
		Language      string    `json:"language"`
		PublicationID uuid.UUID `json:"publication_id"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "topic is required")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	dc, err := h.svc.CheckDuplicate(ctx.Request.Context(), siteID, req.Topic, req.Content, lang, req.PublicationID, time.Now().UTC())
	if err != nil {
		h.log.Error("dedup check failed", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "dedup check failed")
		return
	}
	ctx.JSON(http.StatusOK, dc)
}

// Sweep runs the once-per-day freshness re-evaluation over published articles.
func (h *Handler) Sweep(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Articles []ArticleForSweep `json:"articles"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	decisions, err := h.svc.RunDailySweep(ctx.Request.Context(), siteID, req.Articles)
	if err != nil {
		if errors.Is(err, ErrSweepAlreadyRun) {
			ctx.Error(http.StatusConflict, "SWEEP_ALREADY_RUN", "freshness sweep already ran today")
			return
		}
		h.log.Error("sweep failed", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "sweep failed")
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"decisions": decisions})
}

// Updates lists the Needs Update flags.
func (h *Handler) Updates(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	updates, err := h.svc.ListUpdates(ctx.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list updates", "error", err)
		ctx.Error(http.StatusInternalServerError, "INTERNAL", "failed to list updates")
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"updates": updates})
}

// RegisterRoutes wires the freshness endpoints under /freshness.
func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)
	r.Route("/freshness", func(r chi.Router) {
		r.Post("/classify", rest.AdaptHandler(h.Classify))
		r.Post("/score", rest.AdaptHandler(h.Score))
		r.Post("/obsolete", rest.AdaptHandler(h.Obsolete))
		r.Post("/versions", rest.AdaptHandler(h.SaveVersion))
		r.Get("/versions/{publicationID}", rest.AdaptHandler(h.ListVersions))
		r.Post("/dedup", rest.AdaptHandler(h.Dedup))
		r.Post("/sweep", rest.AdaptHandler(h.Sweep))
		r.Get("/updates", rest.AdaptHandler(h.Updates))
	})
}