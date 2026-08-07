package freshness

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func newTestSvc(t *testing.T, m pgxmock.PgxPoolIface) *Service {
	t.Helper()
	cfg := &config.Config{}
	cfg.Freshness = config.FreshnessConfig{
		SweepEnabled:       true,
		NewsMaxDays:        30,
		NewsNeverOlderDays: 90,
	}
	log := logger.New(cfg)
	db := &database.Database{Pool: m}
	return NewService(cfg, log, db)
}

func TestClassifyAndWindowCaches(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	m.ExpectExec(`INSERT INTO news_intents`).
		WithArgs(siteID, "GPT-6 novo lançamento", pgxmock.AnyArg(), "pt", "update", pgxmock.AnyArg(), pgxmock.AnyArg(), 30, 90, 0).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ir, win, err := svc.ClassifyAndWindow(context.Background(), siteID, "GPT-6 novo lançamento", "", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentUpdate {
		t.Errorf("expected update intent, got %s", ir.Intent)
	}
	if win.VersionFirst != true {
		t.Errorf("expected version-first window, got %+v", win)
	}
}

func TestClassifyAndWindowNoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	ir, win, err := svc.ClassifyAndWindow(context.Background(), uuid.New(), "Empresa anuncia hoje", "", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentNews {
		t.Errorf("expected news, got %s", ir.Intent)
	}
	if win.MaxDays != 30 {
		t.Errorf("expected news max days 30, got %d", win.MaxDays)
	}
}

func TestScoreSourcesPersists(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	now := time.Now().UTC()
	pub := now.AddDate(0, 0, -2)
	upd := now.AddDate(0, 0, -1)

	m.ExpectExec(`INSERT INTO source_freshness_scores`).
		WithArgs(siteID, pgxmock.AnyArg(), "https://reuters.com/tech", "news", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), false, true, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	scored, err := svc.ScoreSources(context.Background(), siteID, nil, IntentNews, "", nil,
		[]SourceInput{{Title: "Reuters", URL: "https://reuters.com/tech", PublishedAt: &pub, UpdatedAt: &upd}})
	if err != nil {
		t.Fatal(err)
	}
	if len(scored) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scored))
	}
	if !scored[0].Usable {
		t.Error("expected fresh source usable")
	}
	if scored[0].SourcePriority != PriorityNewsAgency {
		t.Errorf("expected agency priority, got %s", scored[0].SourcePriority)
	}
}

func TestScoreSourcesObsoleteMarked(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	m.ExpectExec(`INSERT INTO source_freshness_scores`).
		WithArgs(siteID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), true, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	scored, err := svc.ScoreSources(context.Background(), siteID, nil, IntentNews, "", []EntityVersion{{Entity: "GPT", Current: "6"}},
		[]SourceInput{{Title: "guia do GPT-4", URL: "https://example.com/gpt4", Text: "Tutorial do GPT-4 Turbo"}})
	if err != nil {
		t.Fatal(err)
	}
	if !scored[0].Obsolete {
		t.Error("expected source marked obsolete")
	}
}

func TestScoreSourcesRequiresInput(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	_, err := svc.ScoreSources(context.Background(), uuid.New(), nil, IntentNews, "", nil, nil)
	if err == nil {
		t.Error("expected error for empty sources")
	}
}

func TestSaveAndListVersions(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	pubID := uuid.New()

	m.ExpectExec(`INSERT INTO publication_versions`).
		WithArgs(siteID, pubID, "v2", "update", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = svc.SaveVersion(context.Background(), siteID, VersionRecord{
		PublicationID: pubID, Version: "v2", Intent: IntentUpdate,
		Changes: []string{"preço atualizado"}, Diff: []VersionDiff{{Facet: FacetPrice, Before: "20", After: "25", Changed: true}},
		Sources: []string{"reuters.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows := pgxmock.NewRows([]string{"version", "intent", "changes", "diff", "sources", "created_at"}).
		AddRow("v2", "update", []byte(`["preço atualizado"]`), []byte(`[{"facet":"price","before":"20","after":"25","changed":true}]`), []byte(`["reuters.com"]`), time.Now().UTC())
	m.ExpectQuery(`SELECT version, intent, changes, diff, sources, created_at`).
		WithArgs(siteID, pubID).
		WillReturnRows(rows)

	versions, err := svc.ListVersions(context.Background(), siteID, pubID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].Version != "v2" {
		t.Errorf("unexpected versions: %+v", versions)
	}
	if len(versions[0].Diff) != 1 || versions[0].Diff[0].Changed != true {
		t.Errorf("diff not decoded: %+v", versions[0].Diff)
	}
}

func TestCheckDuplicateRegisters(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	rows := pgxmock.NewRows([]string{"publication_id", "topic", "created_on"})
	m.ExpectQuery(`SELECT publication_id, topic, created_on FROM news_dedup`).
		WithArgs(siteID).
		WillReturnRows(rows)

	m.ExpectExec(`INSERT INTO news_dedup`).
		WithArgs(siteID, pgxmock.AnyArg(), "GPT-6 lançamento", "pt", "update", nil, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	dc, err := svc.CheckDuplicate(context.Background(), siteID, "GPT-6 lançamento", "conteúdo", "pt", uuid.Nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if dc.Duplicate {
		t.Error("expected no duplicate on first registration")
	}
	if dc.Fingerprint == "" {
		t.Error("expected fingerprint")
	}
}

func TestCheckDuplicateFindsMatch(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	now := time.Now().UTC()

	rows := pgxmock.NewRows([]string{"publication_id", "topic", "created_on"}).
		AddRow(uuid.New(), "Empresa anuncia novo modelo de IA", now)
	m.ExpectQuery(`SELECT publication_id, topic, created_on FROM news_dedup`).
		WithArgs(siteID).
		WillReturnRows(rows)

	dc, err := svc.CheckDuplicate(context.Background(), siteID, "Empresa anuncia novo modelo de IA", "A empresa anunciou hoje um novo modelo de IA", "pt", uuid.New(), now)
	if err != nil {
		t.Fatal(err)
	}
	if !dc.Duplicate || !dc.SameDay {
		t.Errorf("expected duplicate same-day, got %+v", dc)
	}
}

func TestRunDailySweepMarksUpdates(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	now := time.Now().UTC()

	// no previous sweep row → allowed
	m.ExpectQuery(`SELECT last_run_at FROM freshness_sweeps`).
		WithArgs(siteID).
		WillReturnError(pgx.ErrNoRows)

	// no newer dedup for the article topic
	noRows := pgxmock.NewRows([]string{"1"})
	m.ExpectQuery(`SELECT 1 FROM news_dedup`).
		WithArgs(siteID, pgxmock.AnyArg(), "tema").
		WillReturnRows(noRows)

	// old news article (60 days) → needs update inserted
	m.ExpectExec(`INSERT INTO content_updates`).
		WithArgs(siteID, pgxmock.AnyArg(), "news", "outside_temporal_window", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "needs_update").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// sweep guard recorded
	m.ExpectExec(`INSERT INTO freshness_sweeps`).
		WithArgs(siteID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	decisions, err := svc.RunDailySweep(context.Background(), siteID, []ArticleForSweep{{
		PublicationID: uuid.New(),
		Topic:         "tema",
		PublishedAt:   now.AddDate(0, 0, -60),
		Intent:        IntentNews,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || !decisions[0].NeedsUpdate {
		t.Errorf("expected needs-update decision, got %+v", decisions)
	}
}

func TestRunDailySweepBlockedSameDay(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	now := time.Now().UTC()

	rows := pgxmock.NewRows([]string{"last_run_at"}).AddRow(now)
	m.ExpectQuery(`SELECT last_run_at FROM freshness_sweeps`).
		WithArgs(siteID).
		WillReturnRows(rows)

	_, err = svc.RunDailySweep(context.Background(), siteID, nil)
	if err == nil {
		t.Error("expected sweep blocked when already run today")
	}
}

func TestRunDailySweepDisabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Freshness.SweepEnabled = false
	svc := NewService(cfg, logger.New(cfg), &database.Database{})
	_, err := svc.RunDailySweep(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Error("expected disabled sweep error")
	}
}

func TestListUpdates(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	rows := pgxmock.NewRows([]string{"id", "publication_id", "intent", "reason", "old_score", "new_score", "details", "status", "created_at", "resolved_at"}).
		AddRow(uuid.New(), uuid.New(), "news", "newer_source_found", 80.0, 55.0, []byte(`["nova fonte"]`), "needs_update", time.Now().UTC(), nil)
	m.ExpectQuery(`SELECT id, publication_id, intent, reason, old_score, new_score, details, status, created_at, resolved_at`).
		WithArgs(siteID).
		WillReturnRows(rows)

	updates, err := svc.ListUpdates(context.Background(), siteID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].Status != UpdateNeedsWork {
		t.Errorf("unexpected updates: %+v", updates)
	}
	if len(updates[0].Details) != 1 {
		t.Errorf("details not decoded: %+v", updates[0].Details)
	}
}