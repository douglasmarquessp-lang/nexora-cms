package seoengine

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func newTestSvc(t *testing.T, m pgxmock.PgxPoolIface) *Service {
	t.Helper()
	cfg := &config.Config{}
	cfg.SEO = config.SEOConfig{
		MinPublishScore:            80,
		CompetitorDomains:          []string{"rival.com"},
		InternalLinkMinScore:       40,
		InternalLinkMax:            3,
		ExternalLinkMinReliability: 75,
	}
	log := logger.New(cfg)
	db := &database.Database{Pool: m}
	svc := NewService(cfg, log, db, nil)
	return svc
}

func TestSelectInternalLinks_FiltersByScore(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)

	siteID := uuid.New()
	rows := pgxmock.NewRows([]string{"id", "title", "slug"}).
		AddRow(uuid.New(), "Inteligência artificial para iniciantes", "ia-iniciantes").
		AddRow(uuid.New(), "Receita de bolo de chocolate", "bolo-chocolate")
	m.ExpectQuery(`SELECT id, COALESCE\(title,''\), COALESCE\(slug,''\) FROM posts`).
		WithArgs(siteID, pgxmock.AnyArg()).
		WillReturnRows(rows)

	links, err := svc.SelectInternalLinks(context.Background(), siteID, nil, "Guia de inteligência artificial", "Conteúdo sobre IA", "inteligência artificial", "Tecnologia", 40, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) == 0 {
		t.Fatal("expected at least one link above threshold")
	}
	for _, l := range links {
		if l.Slug == "ia-iniciantes" {
			if l.Relevance < 40 {
				t.Errorf("expected relevant IA post, got %f", l.Relevance)
			}
		}
	}
}

func TestSelectInternalLinks_MaxCap(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)

	siteID := uuid.New()
	rows := pgxmock.NewRows([]string{"id", "title", "slug"})
	for i := 0; i < 5; i++ {
		rows.AddRow(uuid.New(), "Inteligência artificial guia", "ia-guia")
	}
	m.ExpectQuery(`SELECT id, COALESCE\(title,''\), COALESCE\(slug,''\) FROM posts`).
		WithArgs(siteID, pgxmock.AnyArg()).
		WillReturnRows(rows)

	links, err := svc.SelectInternalLinks(context.Background(), siteID, nil, "IA", "", "inteligência artificial", "", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) > 2 {
		t.Errorf("expected max 2 links, got %d", len(links))
	}
}

func TestSelectInternalLinks_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	_, err := svc.SelectInternalLinks(context.Background(), uuid.New(), nil, "", "", "", "", 0, 0)
	if err == nil {
		t.Error("expected error when DB unavailable")
	}
}

func TestSelectExternalLinks_OnlyHighReliability(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)

	siteID := uuid.New()
	rows := pgxmock.NewRows([]string{"url", "title", "domain", "reliability_score"}).
		AddRow("https://www.gov.br/pesquisa", "Estudo oficial", "gov.br", 95).
		AddRow("https://rival.com/blog", "Blog do concorrente", "rival.com", 90).
		AddRow("https://unknown.net/page", "Blog desconhecido", "unknown.net", 10)
	m.ExpectQuery(`SELECT COALESCE\(url,''\), COALESCE\(title,''\), COALESCE\(domain,''\), COALESCE\(reliability_score,0\) FROM research_sources`).
		WithArgs(siteID).
		WillReturnRows(rows)

	links, err := svc.SelectExternalLinks(context.Background(), siteID, "estudo", 70, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range links {
		if l.Domain == "unknown.net" {
			t.Error("expected low-reliability domain to be filtered out")
		}
		if l.Domain == "rival.com" {
			t.Error("expected competitor domain to be filtered out")
		}
		if l.Domain == "gov.br" {
			return
		}
	}
	t.Error("expected the official gov.br source to pass")
}

func TestTopicAuthority_CountsRelated(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)

	siteID := uuid.New()
	rows := pgxmock.NewRows([]string{"title", "slug"}).
		AddRow("Guia de inteligência artificial 2026", "guia-ia").
		AddRow("IA para negócios", "ia-negocios").
		AddRow("Receita de pão", "receita-pao").
		AddRow("Inteligência artificial na saúde", "ia-saude")
	m.ExpectQuery(`SELECT COALESCE\(title,''\), COALESCE\(slug,''\) FROM posts`).
		WithArgs(siteID).
		WillReturnRows(rows)

	report, err := svc.TopicAuthority(context.Background(), siteID, "inteligência artificial")
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage < 2 {
		t.Errorf("expected at least 2 related articles, got %d", report.Coverage)
	}
	if report.Authority <= 0 {
		t.Errorf("expected positive authority, got %f", report.Authority)
	}
}

func TestEnhanceBeforePublish_NoDBContentUnchanged(t *testing.T) {
	svc := NewService(nil, logger.New(&config.Config{}), nil, nil)
	svc.internalLinkMax = 3
	svc.internalLinkMinScore = 40
	svc.externalLinkMinReliability = 75

	enh, err := svc.EnhanceBeforePublish(context.Background(), publisher.ContentEnhancerInput{
		SiteID:   uuid.New(),
		Title:    "Título teste",
		Content:  "Conteúdo para publicação.",
		Language: "pt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if enh == nil || enh.Content != "Conteúdo para publicação." {
		t.Errorf("expected content unchanged (no DB), got %q", enh.Content)
	}
	if enh.GapReport == nil {
		t.Error("expected gap report even without DB")
	}
}

func TestEnhanceBeforePublish_NilService(t *testing.T) {
	var svc *Service
	enh, err := svc.EnhanceBeforePublish(context.Background(), publisher.ContentEnhancerInput{})
	if err != nil || enh != nil {
		t.Errorf("expected nil enhancer result for nil service, got %v / %v", enh, err)
	}
}