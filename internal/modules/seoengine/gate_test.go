package seoengine

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestCheckPublishScore_StoredScore(t *testing.T) {
	svc, mock := setupMockDB(t)
	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT seo_score FROM posts WHERE`).
		WithArgs(&postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"seo_score"}).AddRow(92.5))

	score, err := svc.CheckPublishScore(context.Background(), publisher.PublishGateInput{
		SiteID:   siteID,
		PostID:   &postID,
		Title:    "T",
		Content:  "C",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 92.5 {
		t.Errorf("expected stored score 92.5, got %f", score)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestCheckPublishScore_InlineFallbackNoRow(t *testing.T) {
	svc, mock := setupMockDB(t)
	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT seo_score FROM posts WHERE`).
		WithArgs(postID, siteID).
		WillReturnError(pgx.ErrNoRows)

	score, err := svc.CheckPublishScore(context.Background(), publisher.PublishGateInput{
		SiteID:   siteID,
		PostID:   &postID,
		Title:    "Guia de Marketing de Conteúdo",
		Content:  "# Guia de Marketing\nMarketing de conteúdo é o tema deste guia longo e completo sobre estratégias.",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score < 0 || score > 100 {
		t.Errorf("expected score in [0,100], got %f", score)
	}
}

func TestCheckPublishScore_InlineFallbackDBError(t *testing.T) {
	svc, mock := setupMockDB(t)
	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT seo_score FROM posts WHERE`).
		WithArgs(postID, siteID).
		WillReturnError(pgx.ErrTxClosed)

	score, err := svc.CheckPublishScore(context.Background(), publisher.PublishGateInput{
		SiteID:   siteID,
		PostID:   &postID,
		Title:    "Guia de Marketing de Conteúdo",
		Content:  "conteúdo simples",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("expected fail-open, got %v", err)
	}
	if score < 0 || score > 100 {
		t.Errorf("expected score in [0,100], got %f", score)
	}
}

func TestCheckPublishScore_NoPostInline(t *testing.T) {
	svc, _ := setupMockDB(t)

	score, err := svc.CheckPublishScore(context.Background(), publisher.PublishGateInput{
		SiteID:   uuid.New(),
		Title:    "Guia de Marketing de Conteúdo",
		Content:  "conteúdo simples",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score < 0 || score > 100 {
		t.Errorf("expected score in [0,100], got %f", score)
	}
}

func TestCheckPublishScore_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	score, err := svc.CheckPublishScore(context.Background(), publisher.PublishGateInput{
		SiteID:   uuid.New(),
		Title:    "Guia de Marketing de Conteúdo",
		Content:  "conteúdo simples",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("expected no error without DB, got %v", err)
	}
	if score < 0 || score > 100 {
		t.Errorf("expected score in [0,100], got %f", score)
	}
}

func TestCheckPublishScore_Deterministic(t *testing.T) {
	svc, _ := setupMockDB(t)
	in := publisher.PublishGateInput{
		SiteID:   uuid.New(),
		Title:    "Guia de Marketing de Conteúdo",
		Content:  "conteúdo simples para teste",
		Language: "pt",
	}
	a, _ := svc.CheckPublishScore(context.Background(), in)
	b, _ := svc.CheckPublishScore(context.Background(), in)
	if a != b {
		t.Errorf("expected deterministic scores, got %f vs %f", a, b)
	}
}
