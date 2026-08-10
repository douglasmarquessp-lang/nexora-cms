package publisher

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type fakeGate struct {
	score float64
	err   error
	got   *PublishGateInput
}

func (f *fakeGate) CheckPublishScore(ctx context.Context, in PublishGateInput) (float64, error) {
	f.got = &in
	return f.score, f.err
}

func gateSvc(minScore float64, gate PublishGate) *Service {
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = minScore
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	svc.SetPublishGate(gate)
	return svc
}

func TestCheckPublishGate_Disabled(t *testing.T) {
	ctx := context.Background()

	svc := gateSvc(0, &fakeGate{score: 10})
	if err := svc.checkPublishGate(ctx, uuid.New(), nil, "T", "C", "pt", ""); err != nil {
		t.Errorf("expected no gate check when min score is 0, got %v", err)
	}

	svc = gateSvc(80, nil)
	if err := svc.checkPublishGate(ctx, uuid.New(), nil, "T", "C", "pt", ""); err != nil {
		t.Errorf("expected no gate check without gate, got %v", err)
	}
}

func TestCheckPublishGate_EmptyContent(t *testing.T) {
	ctx := context.Background()
	fg := &fakeGate{score: 10}
	svc := gateSvc(80, fg)
	if err := svc.checkPublishGate(ctx, uuid.New(), nil, "", "", "pt", ""); err != nil {
		t.Errorf("expected skip for empty title/content, got %v", err)
	}
	if fg.got != nil {
		t.Error("gate must not be consulted for empty content")
	}
}

func TestCheckPublishGate_BlocksLowScore(t *testing.T) {
	ctx := context.Background()
	fg := &fakeGate{score: 55}
	svc := gateSvc(80, fg)

	err := svc.checkPublishGate(ctx, uuid.New(), nil, "Titulo", "Conteúdo", "pt", "")
	if !errors.Is(err, ErrSEOPublishBlocked) {
		t.Errorf("expected ErrSEOPublishBlocked, got %v", err)
	}
	if fg.got == nil || fg.got.Title != "Titulo" || fg.got.Language != "pt" {
		t.Errorf("unexpected gate input: %+v", fg.got)
	}
}

func TestCheckPublishGate_AllowsHighScore(t *testing.T) {
	ctx := context.Background()
	postID := uuid.New()
	fg := &fakeGate{score: 95}
	svc := gateSvc(80, fg)

	if err := svc.checkPublishGate(ctx, uuid.New(), &postID, "Titulo", "Conteúdo", "pt", ""); err != nil {
		t.Errorf("expected publish allowed, got %v", err)
	}
	if fg.got == nil || fg.got.PostID == nil || *fg.got.PostID != postID {
		t.Errorf("expected post id forwarded to gate, got %+v", fg.got)
	}
}

func TestCheckPublishGate_FailsOpenOnGateError(t *testing.T) {
	ctx := context.Background()
	fg := &fakeGate{err: errors.New("boom")}
	svc := gateSvc(80, fg)

	if err := svc.checkPublishGate(ctx, uuid.New(), nil, "Titulo", "Conteúdo", "pt", ""); err != nil {
		t.Errorf("expected fail-open on gate error, got %v", err)
	}
}

func TestCheckPublishGate_ForwardsMetaDescription(t *testing.T) {
	ctx := context.Background()
	fg := &fakeGate{score: 95}
	svc := gateSvc(80, fg)

	err := svc.checkPublishGate(ctx, uuid.New(), nil, "Titulo", "Conteúdo", "pt", "Meta descrição do artigo")
	if err != nil {
		t.Fatalf("expected gate pass, got %v", err)
	}
	if fg.got == nil || fg.got.MetaDescription != "Meta descrição do artigo" {
		t.Errorf("expected MetaDescription forwarded to gate, got %+v", fg.got)
	}
}

func TestPublishGeneratedArticle_BlockedBeforeDB(t *testing.T) {
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = 80
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	svc.SetPublishGate(&fakeGate{score: 40})

	_, err := svc.PublishGeneratedArticle(context.Background(), PublishGeneratedRequest{
		SiteID:   uuid.New(),
		Title:    "Artigo com SEO ruim",
		Content:  "conteúdo curto",
		Language: "pt",
	})
	if !errors.Is(err, ErrSEOPublishBlocked) {
		t.Errorf("expected ErrSEOPublishBlocked before any DB access, got %v", err)
	}
}

func TestPublishArticle_NoDBUnchangedWithoutGate(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, &database.Database{}, nil)

	_, err := svc.PublishArticle(context.Background(), uuid.New(), uuid.New(), PublishRequest{
		Title:    "Teste",
		Language: "pt",
	})
	if err == nil {
		t.Error("expected DB error without gate configured")
	}
}

type fakeEnhancer struct {
	out   *ContentEnhancement
	err   error
	gotIn *ContentEnhancerInput
}

func (f *fakeEnhancer) EnhanceBeforePublish(ctx context.Context, in ContentEnhancerInput) (*ContentEnhancement, error) {
	f.gotIn = &in
	if f.err != nil {
		return nil, f.err
	}
	if f.out == nil {
		return &ContentEnhancement{Content: in.Content}, nil
	}
	return f.out, nil
}

func TestEnhanceContent_NilEnhancer(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	out, _, _ := svc.enhanceContent(context.Background(), uuid.New(), nil, "T", "conteúdo original", "kw", "cat", "pt", "", "")
	if out != "conteúdo original" {
		t.Errorf("expected unchanged content, got %q", out)
	}
}

func TestEnhanceContent_EmptyContent(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	svc.SetContentEnhancer(&fakeEnhancer{out: &ContentEnhancement{Content: "enhanced"}})
	if out, _, _ := svc.enhanceContent(context.Background(), uuid.New(), nil, "T", "", "", "", "pt", "", ""); out != "" {
		t.Errorf("expected empty content unchanged, got %q", out)
	}
}

func TestEnhanceContent_AppliesResult(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	enh := &fakeEnhancer{out: &ContentEnhancement{Content: "conteúdo enriquecido com links"}}
	svc.SetContentEnhancer(enh)

	out, _, _ := svc.enhanceContent(context.Background(), uuid.New(), nil, "Titulo", "conteúdo original", "kw", "cat", "pt", "", "")
	if out != "conteúdo enriquecido com links" {
		t.Errorf("expected enhanced content, got %q", out)
	}
	if enh.gotIn == nil {
		t.Fatal("expected enhancer input")
	}
	if enh.gotIn.SiteID == uuid.Nil {
		t.Error("expected site id propagated")
	}
	if enh.gotIn.Keyword != "kw" || enh.gotIn.Category != "cat" || enh.gotIn.Language != "pt" {
		t.Errorf("unexpected input fields: %+v", enh.gotIn)
	}
}

func TestEnhanceContent_FailsOpen(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	svc.SetContentEnhancer(&fakeEnhancer{err: errors.New("enhancement boom")})
	out, _, _ := svc.enhanceContent(context.Background(), uuid.New(), nil, "T", "conteúdo original", "", "", "pt", "", "")
	if out != "conteúdo original" {
		t.Errorf("expected original content on enhancer error, got %q", out)
	}
}

func TestEnhanceContent_EmptyResultFallsBack(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	svc.SetContentEnhancer(&fakeEnhancer{out: &ContentEnhancement{Content: ""}})
	out, _, _ := svc.enhanceContent(context.Background(), uuid.New(), nil, "T", "conteúdo original", "", "", "pt", "", "")
	if out != "conteúdo original" {
		t.Errorf("expected original content when enhancer returns empty, got %q", out)
	}
}

func TestPublishGeneratedArticle_EnhancerRuns(t *testing.T) {
	// Without DB + without gate, the enhancer must run and its output must be
	// what is passed on; publishing then fails at DB (repository nil-safe error),
	// but the enhancer input must carry the right content.
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = 0
	log := logger.New(cfg)
	svc := NewService(cfg, log, &database.Database{}, nil)
	enh := &fakeEnhancer{out: &ContentEnhancement{Content: "conteúdo enriquecido"}}
	svc.SetContentEnhancer(enh)

	_, err := svc.PublishGeneratedArticle(context.Background(), PublishGeneratedRequest{
		SiteID:   uuid.New(),
		Title:    "Título",
		Content:  "conteúdo original",
		Language: "pt",
	})
	if err == nil {
		t.Error("expected DB error (no repo), but publishing must attempt after enhancement")
	}
	if enh.gotIn == nil {
		t.Fatal("expected enhancer to run")
	}
	if enh.gotIn.Content != "conteúdo original" {
		t.Errorf("expected enhancer to receive original content, got %q", enh.gotIn.Content)
	}
}
