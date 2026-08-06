package editorialbrain

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

type harness struct {
	router chi.Router
}

func newHarness(t *testing.T, svc *Service) *harness {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.CtxSiteID, uuid.New())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	RegisterRoutes(router, svc, log)
	return &harness{router: router}
}

func do(t *testing.T, h *harness, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func TestIntentEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/intent", map[string]interface{}{
		"topic":    "Gemini 3 lançado hoje: anúncio oficial",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out IntentResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Intent != IntentBreakingNews {
		t.Errorf("expected breaking_news, got %s", out.Intent)
	}
}

func TestIntentEndpointValidation(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/intent", map[string]interface{}{"topic": ""})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty topic, got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/editorialbrain/intent", map[string]interface{}{"topic": "x", "language": "fr"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid language, got %d", rec.Code)
	}
}

func TestPersonaEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/persona", map[string]interface{}{
		"topic":    "gemini para criadores de conteúdo no youtube",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out PersonaResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Persona != PersonaCreator {
		t.Errorf("expected creator, got %s", out.Persona)
	}
}

func TestOutlineEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/outline", map[string]interface{}{
		"topic": "como usar o gemini",
		"intent": "tutorial",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out EditorialOutline
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Sections) == 0 {
		t.Error("expected outline sections")
	}
}

func TestQuestionsEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/questions", map[string]interface{}{
		"topic":  "o gemini 3",
		"intent": "commercial",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Questions []RequiredQuestion `json:"questions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Questions) == 0 {
		t.Error("expected required questions")
	}
}

func TestCoverageEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/coverage", map[string]interface{}{
		"content": "## O que é\n\nO Gemini é uma IA. ## Preço\n\nCusta US$ 20 por mês.",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out CoverageReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.CoveragePercent < 0 || out.CoveragePercent > 100 {
		t.Errorf("coverage out of range: %v", out.CoveragePercent)
	}
}

func TestFluencyEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/fluency", map[string]interface{}{
		"content": "O Gemini foi lançado em 2026 e é rápido. Ele custa US$ 20 por mês.",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out FluencyReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.OverallScore <= 0 || out.OverallScore > 100 {
		t.Errorf("fluency out of range: %v", out.OverallScore)
	}
}

func TestEvidenceEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/evidence", map[string]interface{}{
		"content": "O Gemini 3 foi lançado em 2026.",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out EvidenceReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ClaimsCount != 1 || len(out.Links) != 1 {
		t.Errorf("expected 1 claim, got %d / %d", out.ClaimsCount, len(out.Links))
	}
}

func TestEvidenceEndpointInvalidJob(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/evidence", map[string]interface{}{
		"content": "O Gemini 3 foi lançado em 2026.", "research_job_id": "not-a-uuid",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid research_job_id, got %d", rec.Code)
	}
}

func TestSemanticEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/semantic", map[string]interface{}{
		"topic": "O Gemini 3", "content": "O Gemini 3 é rápido e custa US$ 20 por mês.",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out SemanticReport
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.SemanticScore <= 0 || out.SemanticScore > 100 {
		t.Errorf("semantic out of range: %v", out.SemanticScore)
	}
}

func TestScoreEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/score", map[string]interface{}{
		"seo": 96, "eeat": 94, "freshness": 98, "coverage": 95, "naturalness": 97, "confidence": 99,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out EditorialScore
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Decision != DecisionApproved || out.Threshold != 90 {
		t.Errorf("expected approved @90, got %s @%v", out.Decision, out.Threshold)
	}
}

func TestCreateBriefEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/brief", map[string]interface{}{
		"topic": "O que é o Gemini 3?", "language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out EditorialBrief
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.SearchIntent != IntentInformational {
		t.Errorf("expected informational, got %s", out.SearchIntent)
	}
}

func TestCreateReviewEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/editorialbrain/review", map[string]interface{}{
		"title": "O Gemini 3", "content": "O Gemini 3 foi lançado em 2026.",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out EditorialReview
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Scores.Final <= 0 {
		t.Errorf("expected final score, got %+v", out.Scores)
	}
}

func TestMissingSiteEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	cfg := &config.Config{}
	log := logger.New(cfg)
	router := chi.NewRouter()
	RegisterRoutes(router, svc, log)

	body := bytes.NewBufferString(`{"topic":"gemini"}`)
	req := httptest.NewRequest("POST", "/editorialbrain/intent", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 MISSING_SITE, got %d", rec.Code)
	}
	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Error.Code != "MISSING_SITE" {
		t.Errorf("expected MISSING_SITE, got %s", out.Error.Code)
	}
}
