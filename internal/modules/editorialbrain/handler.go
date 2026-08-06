package editorialbrain

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/pkg/logger"
)

// Handler exposes the editorial brain REST endpoints.
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

// RegisterRoutes wires all /editorialbrain endpoints.
func RegisterRoutes(r chi.Router, svc *Service, log *logger.Logger) {
	h := NewHandler(svc, log)
	r.Post("/editorialbrain/intent", rest.AdaptHandler(h.Intent))
	r.Post("/editorialbrain/persona", rest.AdaptHandler(h.Persona))
	r.Post("/editorialbrain/outline", rest.AdaptHandler(h.Outline))
	r.Post("/editorialbrain/questions", rest.AdaptHandler(h.Questions))
	r.Post("/editorialbrain/coverage", rest.AdaptHandler(h.Coverage))
	r.Post("/editorialbrain/fluency", rest.AdaptHandler(h.Fluency))
	r.Post("/editorialbrain/evidence", rest.AdaptHandler(h.Evidence))
	r.Post("/editorialbrain/semantic", rest.AdaptHandler(h.Semantic))
	r.Post("/editorialbrain/score", rest.AdaptHandler(h.Score))
	r.Post("/editorialbrain/brief", rest.AdaptHandler(h.CreateBrief))
	r.Get("/editorialbrain/briefs", rest.AdaptHandler(h.ListBriefs))
	r.Get("/editorialbrain/brief/{id}", rest.AdaptHandler(h.GetBrief))
	r.Post("/editorialbrain/review", rest.AdaptHandler(h.CreateReview))
	r.Get("/editorialbrain/reviews", rest.AdaptHandler(h.ListReviews))
	r.Get("/editorialbrain/review/{id}", rest.AdaptHandler(h.GetReview))
}

// Intent classifies the search intent of a topic.
func (h *Handler) Intent(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic    string `json:"topic"`
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
	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "TOPIC_REQUIRED", "topic is required")
		return
	}
	if lang != "pt" && lang != "en" {
		ctx.Error(http.StatusBadRequest, "INVALID_LANGUAGE", "language must be pt or en")
		return
	}
	ctx.JSON(http.StatusOK, ClassifyIntent(req.Topic, lang))
}

// Persona detects the most likely reader persona of a topic.
func (h *Handler) Persona(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic    string `json:"topic"`
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
	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "TOPIC_REQUIRED", "topic is required")
		return
	}
	ctx.JSON(http.StatusOK, DetectPersona(req.Topic, lang))
}

// Outline generates the intelligent outline for a topic.
func (h *Handler) Outline(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic    string       `json:"topic"`
		Language string       `json:"language"`
		Intent   SearchIntent `json:"intent"`
		Persona  Persona      `json:"persona"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "TOPIC_REQUIRED", "topic is required")
		return
	}
	if req.Intent == "" {
		req.Intent = IntentInformational
	}
	if req.Persona == "" {
		req.Persona = PersonaGeneral
	}
	ctx.JSON(http.StatusOK, GenerateOutline(req.Topic, lang, req.Intent, req.Persona))
}

// Questions builds the required question list (with answer verification
// when `text` is provided).
func (h *Handler) Questions(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic    string       `json:"topic"`
		Language string       `json:"language"`
		Intent   SearchIntent `json:"intent"`
		Text     string       `json:"text"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if req.Topic == "" {
		ctx.Error(http.StatusBadRequest, "TOPIC_REQUIRED", "topic is required")
		return
	}
	if req.Intent == "" {
		req.Intent = IntentInformational
	}
	questions := GenerateQuestions(req.Topic, lang, req.Intent)
	if req.Text != "" {
		ctx.JSON(http.StatusOK, CheckQuestions(req.Text, questions))
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"questions": questions})
}

// Coverage measures how much of the subject the article explains.
func (h *Handler) Coverage(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
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
	if req.Content == "" {
		ctx.Error(http.StatusBadRequest, "CONTENT_REQUIRED", "content is required")
		return
	}
	ctx.JSON(http.StatusOK, CheckCoverage(req.Content, lang))
}

// Fluency checks reading fluency of the article.
func (h *Handler) Fluency(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
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
	if req.Content == "" {
		ctx.Error(http.StatusBadRequest, "CONTENT_REQUIRED", "content is required")
		return
	}
	ctx.JSON(http.StatusOK, CheckFluency(ctx.Request.Context(), req.Content, lang, h.svc.qualityChecker))
}

// Evidence links important claims to the research fact base.
func (h *Handler) Evidence(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Content       string     `json:"content"`
		Language      string     `json:"language"`
		ResearchJobID string     `json:"research_job_id"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if req.Content == "" {
		ctx.Error(http.StatusBadRequest, "CONTENT_REQUIRED", "content is required")
		return
	}
	var jobID *uuid.UUID
	if req.ResearchJobID != "" {
		id, err := uuid.Parse(req.ResearchJobID)
		if err != nil {
			ctx.Error(http.StatusBadRequest, "INVALID_RESEARCH_JOB", "invalid research_job_id")
			return
		}
		jobID = &id
	}
	facts, sources := h.svc.loadResearch(ctx.Request.Context(), siteID, jobID, "", lang)
	ctx.JSON(http.StatusOK, LinkEvidence(req.Content, facts, sources, lang))
}

// Semantic checks entities, related concepts, missing terms and synonyms.
func (h *Handler) Semantic(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		Topic         string     `json:"topic"`
		Content       string     `json:"content"`
		Language      string     `json:"language"`
		ResearchJobID string     `json:"research_job_id"`
		Entities      []string   `json:"entities"`
		Concepts      []string   `json:"concepts"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if req.Content == "" {
		ctx.Error(http.StatusBadRequest, "CONTENT_REQUIRED", "content is required")
		return
	}
	facts, _ := h.svc.loadResearch(ctx.Request.Context(), siteID, parseJobPtr(req.ResearchJobID), "", lang)
	_ = facts
	questions := GenerateQuestions(req.Topic, lang, ClassifyIntent(req.Topic, lang).Intent)
	qc := CheckQuestions(req.Content, questions)
	ctx.JSON(http.StatusOK, CheckSemantic(req.Topic, req.Content, lang, req.Entities, req.Concepts, qc.AnsweredPercent))
}

// Score computes the final editorial note for the given components.
func (h *Handler) Score(ctx *rest.Context) {
	_, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		SEO         float64 `json:"seo"`
		EEAT        float64 `json:"eeat"`
		Freshness   float64 `json:"freshness"`
		Coverage    float64 `json:"coverage"`
		Naturalness float64 `json:"naturalness"`
		Confidence  float64 `json:"confidence"`
		Threshold   float64 `json:"threshold"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	threshold := req.Threshold
	if threshold <= 0 {
		threshold = h.svc.MinFinalScore()
	}
	ctx.JSON(http.StatusOK, ComputeEditorialScore(req.SEO, req.EEAT, req.Freshness,
		req.Coverage, req.Naturalness, req.Confidence, threshold))
}

// CreateBrief builds and persists an editorial brief.
func (h *Handler) CreateBrief(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	userID, _ := middleware.GetUserID(ctx.Request.Context())
	var req struct {
		Topic    string `json:"topic"`
		Language string `json:"language"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	brief, err := h.svc.BuildBrief(ctx.Request.Context(), siteID, userID, req.Topic, req.Language)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "BRIEF_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, brief)
}

// ListBriefs lists the site's briefs.
func (h *Handler) ListBriefs(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	limit := queryInt(ctx.Request, "limit", 20)
	offset := queryInt(ctx.Request, "offset", 0)
	briefs, err := h.svc.ListBriefs(ctx.Request.Context(), siteID, limit, offset)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "LIST_BRIEFS_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"briefs": briefs, "total": len(briefs)})
}

// GetBrief loads a brief by id.
func (h *Handler) GetBrief(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	briefID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid brief id")
		return
	}
	brief, err := h.svc.GetBrief(ctx.Request.Context(), siteID, briefID)
	if err != nil {
		if errors.Is(err, ErrBriefNotFound) {
			ctx.Error(http.StatusNotFound, "BRIEF_NOT_FOUND", "brief not found")
			return
		}
		ctx.Error(http.StatusInternalServerError, "GET_BRIEF_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, brief)
}

// CreateReview runs the full editorial note and returns the gate decision.
func (h *Handler) CreateReview(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	var req struct {
		BriefID       string `json:"brief_id"`
		ArticleID     string `json:"article_id"`
		Title         string `json:"title"`
		Content       string `json:"content"`
		Language      string `json:"language"`
		ResearchJobID string `json:"research_job_id"`
	}
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	review, err := h.svc.ReviewArticle(ctx.Request.Context(), siteID, ReviewRequest{
		BriefID:       parseJobPtr(req.BriefID),
		ArticleID:     parseJobPtr(req.ArticleID),
		Title:         req.Title,
		Content:       req.Content,
		Language:      req.Language,
		ResearchJobID: parseJobPtr(req.ResearchJobID),
	})
	if err != nil {
		ctx.Error(http.StatusBadRequest, "REVIEW_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, review)
}

// ListReviews lists the site's reviews, optionally filtered by decision.
func (h *Handler) ListReviews(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	decision := ctx.Request.URL.Query().Get("decision")
	limit := queryInt(ctx.Request, "limit", 20)
	offset := queryInt(ctx.Request, "offset", 0)
	reviews, err := h.svc.ListReviews(ctx.Request.Context(), siteID, decision, limit, offset)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "LIST_REVIEWS_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{"reviews": reviews, "total": len(reviews)})
}

// GetReview loads a review by id.
func (h *Handler) GetReview(ctx *rest.Context) {
	siteID, ok := h.siteID(ctx)
	if !ok {
		return
	}
	reviewID, err := uuid.Parse(chi.URLParam(ctx.Request, "id"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_ID", "invalid review id")
		return
	}
	review, err := h.svc.GetReview(ctx.Request.Context(), siteID, reviewID)
	if err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			ctx.Error(http.StatusNotFound, "REVIEW_NOT_FOUND", "review not found")
			return
		}
		ctx.Error(http.StatusInternalServerError, "GET_REVIEW_ERROR", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, review)
}

// parseJobPtr parses an optional UUID string into a pointer.
func parseJobPtr(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

// queryInt reads an int query param with a default.
func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}
