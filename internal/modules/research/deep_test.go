package research

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestResearchTopicHash(t *testing.T) {
	a := researchTopicHash("GPT-6")
	b := researchTopicHash("gpt-6")
	c := researchTopicHash(" GPT-6 ")
	d := researchTopicHash("Other topic")
	if len(a) != 16 {
		t.Errorf("expected 16-char hash, got %d", len(a))
	}
	if a != b || a != c {
		t.Error("hash must be case-folded and trimmed")
	}
	if a == d {
		t.Error("different topics must produce different hashes")
	}
}

func TestDeepResearch_CacheHit(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM research_cache`).
		WithArgs(siteID, researchTopicHash("GPT-6"), "en").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "briefing", "fact_base", "sources",
			"hit_count", "created_at", "expires_at",
		}).AddRow(
			uuid.New(), siteID, "GPT-6", "en",
			`{"topic":"GPT-6","summary":"cached summary"}`,
			`[{"fact_type":"version","entity":"GPT-6","value":"2.0","confidence":80}]`,
			`[{"title":"OpenAI","url":"https://openai.com","reliability_score":100}]`,
			3, now, now.Add(time.Hour),
		))

	mock.ExpectExec(`UPDATE research_cache SET hit_count`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	report, err := svc.DeepResearch(context.Background(), siteID, "GPT-6", "en")
	if err != nil {
		t.Fatalf("DeepResearch failed: %v", err)
	}
	if !report.Cached {
		t.Error("expected cached=true on cache hit")
	}
	if len(report.Sources) != 1 {
		t.Errorf("expected 1 cached source, got %d", len(report.Sources))
	}
	if len(report.Facts) != 1 {
		t.Errorf("expected 1 cached fact, got %d", len(report.Facts))
	}
	if report.Briefing == nil || report.Briefing.Summary != "cached summary" {
		t.Errorf("cached briefing mismatch: %+v", report.Briefing)
	}
}

func TestDeepResearch_CacheMissFullFlow(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	// 1. Cache lookup → miss.
	mock.ExpectQuery(`SELECT .+ FROM research_cache`).
		WithArgs(siteID, researchTopicHash("GPT-6"), "en").
		WillReturnError(pgx.ErrNoRows)

	// 2. CreateJob (nil AI manager → no grounding sources → no AddSource).
	mock.ExpectExec(`INSERT INTO research_jobs`).
		WithArgs(pgxmock.AnyArg(), siteID, "GPT-6", "en", "", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// 3. persistBriefing → SaveBriefing INSERT + GetBriefingByJobID SELECT.
	mock.ExpectExec(`INSERT INTO research_briefings`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(`SELECT .+ FROM research_briefings WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "research_job_id", "structured_briefing", "timeline", "confirmed_facts",
			"conflicting_info", "editorial_approaches", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), `{"summary":"s"}`, "[]", "[]", "[]", "[]", now, now,
		))

	// 4. saveCache.
	mock.ExpectExec(`INSERT INTO research_cache`).
		WithArgs(pgxmock.AnyArg(), siteID, "GPT-6", pgxmock.AnyArg(), "en",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// 5. CompleteJob → UpdateJob → GetJob + UPDATE.
	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), siteID, "GPT-6", "en", "", "pending", 0, "", nil, now, now,
		))
	mock.ExpectExec(`UPDATE research_jobs SET status = .+`).
		WithArgs("completed", pgxmock.AnyArg(), pgxmock.AnyArg(), siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	report, err := svc.DeepResearch(context.Background(), siteID, "GPT-6", "en")
	if err != nil {
		t.Fatalf("DeepResearch failed: %v", err)
	}
	if report.Cached {
		t.Error("expected cached=false on fresh research")
	}
	if report.ResearchJob.Status != JobStatusPending {
		t.Errorf("report job should keep pre-completion status, got %v", report.ResearchJob.Status)
	}
	if len(report.Sources) != 0 {
		t.Errorf("no AI → no grounding sources expected, got %d", len(report.Sources))
	}
}

func TestDeepResearch_InvalidInputs(t *testing.T) {
	svc, _ := setupMockDB(t)
	ctx := context.Background()
	siteID := uuid.New()

	if _, err := svc.DeepResearch(ctx, siteID, "  ", "en"); !errors.Is(err, ErrTopicRequired) {
		t.Errorf("expected ErrTopicRequired, got %v", err)
	}
	if _, err := svc.DeepResearch(ctx, siteID, "Topic", "fr"); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestDeepResearch_NoDB(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg, logger.New(cfg), nil, nil)
	if _, err := svc.DeepResearch(context.Background(), uuid.New(), "Topic", "en"); err == nil {
		t.Error("expected error without database")
	}
}

func TestGetCachedResearch_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT .+ FROM research_cache`).
		WithArgs(siteID, researchTopicHash("T"), "pt").
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetCachedResearch(context.Background(), siteID, "T", "pt")
	if !errors.Is(err, ErrCacheEntryNotFound) {
		t.Errorf("expected ErrCacheEntryNotFound, got %v", err)
	}
}

func TestGetCachedSummary_NeverTriggersResearch(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	// Only the cache SELECT is expected — if the code tried to create a job or
	// search, the mock would fail on the unexpected query.
	mock.ExpectQuery(`SELECT .+ FROM research_cache`).
		WithArgs(siteID, researchTopicHash("T"), "en").
		WillReturnError(pgx.ErrNoRows)

	if _, err := svc.GetCachedSummary(context.Background(), siteID, "T", "en"); err == nil {
		t.Error("expected error for cache miss")
	}
}

func TestGetCachedSummary_ConvertsEntries(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM research_cache`).
		WithArgs(siteID, researchTopicHash("T"), "en").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "briefing", "fact_base", "sources",
			"hit_count", "created_at", "expires_at",
		}).AddRow(
			uuid.New(), siteID, "T", "en",
			`{"topic":"T","summary":"S","key_points":["K"]}`,
			`[{"fact_type":"company","entity":"OpenAI","value":"OpenAI","confidence":75}]`,
			`[{"title":"Src","url":"https://openai.com","domain":"openai.com","reliability_score":100,"reliability_label":"verified"}]`,
			1, now, now.Add(time.Hour),
		))

	mock.ExpectExec(`UPDATE research_cache SET hit_count`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	summary, err := svc.GetCachedSummary(context.Background(), siteID, "T", "en")
	if err != nil {
		t.Fatalf("GetCachedSummary failed: %v", err)
	}
	if !summary.Cached {
		t.Error("expected cached=true")
	}
	if len(summary.Facts) != 1 || summary.Facts[0].Type != "company" {
		t.Errorf("facts conversion mismatch: %+v", summary.Facts)
	}
	if len(summary.Sources) != 1 || summary.Sources[0].ReliabilityLabel != "verified" {
		t.Errorf("sources conversion mismatch: %+v", summary.Sources)
	}
	if summary.Briefing == "" || !containsStr(summary.Briefing, "Key points") {
		t.Errorf("briefing text missing sections: %q", summary.Briefing)
	}
}

func TestDeepResearchSummary_AlwaysNonNil(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT .+ FROM research_cache`).
		WithArgs(siteID, researchTopicHash("T"), "en").
		WillReturnError(pgx.ErrNoRows)

	mock.ExpectExec(`INSERT INTO research_jobs`).
		WithArgs(pgxmock.AnyArg(), siteID, "T", "en", "", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`INSERT INTO research_briefings`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(`SELECT .+ FROM research_briefings WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "research_job_id", "structured_briefing", "timeline", "confirmed_facts",
			"conflicting_info", "editorial_approaches", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), `{"summary":"s"}`, "[]", "[]", "[]", "[]", now, now))

	mock.ExpectExec(`INSERT INTO research_cache`).
		WithArgs(pgxmock.AnyArg(), siteID, "T", pgxmock.AnyArg(), "en",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(uuid.New(), siteID, "T", "en", "", "pending", 0, "", nil, now, now))
	mock.ExpectExec(`UPDATE research_jobs SET status = .+`).
		WithArgs("completed", pgxmock.AnyArg(), pgxmock.AnyArg(), siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	summary, err := svc.DeepResearchSummary(context.Background(), siteID, "T", "en")
	if err != nil {
		t.Fatalf("DeepResearchSummary failed: %v", err)
	}
	if summary == nil {
		t.Fatal("summary must never be nil")
	}
	if summary.Topic != "T" || summary.Language != "en" {
		t.Errorf("summary fields mismatch: %+v", summary)
	}
}

func TestDecorateSources(t *testing.T) {
	svc, _ := setupMockDB(t)
	sources := svc.decorateSources([]ResearchSource{
		{URL: "https://www.reuters.com/tech"},
		{URL: "https://unknown-blog.xyz/post"},
	})
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].Domain != "reuters.com" || sources[0].ReliabilityScore != 95 || sources[0].ReliabilityLabel != "verified" {
		t.Errorf("reuters decoration wrong: %+v", sources[0])
	}
	if sources[1].Domain != "unknown-blog.xyz" || sources[1].ReliabilityScore != 30 {
		t.Errorf("unknown domain decoration wrong: %+v", sources[1])
	}
}

func TestReportFromCache(t *testing.T) {
	svc, _ := setupMockDB(t)
	_ = svc
	c := &CachedResearch{
		Topic: "T", Language: "en",
		Briefing: ResearchBriefingDoc{Topic: "T", Summary: "S"},
		Facts:    []FactBaseEntry{{FactType: FactTypeDate, Entity: "E", Value: "V"}},
		Sources:  []ResearchSource{{Title: "S1"}},
	}
	report := (&Service{}).reportFromCache(c)
	if !report.Cached || report.ResearchJob.Status != JobStatusCompleted {
		t.Errorf("cached report fields wrong: %+v", report)
	}
	if report.Briefing == nil || report.Briefing.Summary != "S" || len(report.Facts) != 1 || len(report.Sources) != 1 {
		t.Errorf("cached report content wrong: %+v", report)
	}
}

func TestFormatBriefingDoc(t *testing.T) {
	doc := ResearchBriefingDoc{
		Topic: "T", Summary: "S",
		KeyPoints:  []string{"K1"},
		Companies:  []string{"OpenAI"},
		Statistics: []string{"X → 1"},
	}
	out := formatBriefingDoc(doc)
	for _, want := range []string{"Topic: T", "Summary: S", "Key points:", "- K1", "Companies:", "- OpenAI", "Statistics:"} {
		if !containsStr(out, want) {
			t.Errorf("formatted briefing missing %q:\n%s", want, out)
		}
	}
	if containsStr(out, "Products:") {
		t.Error("empty sections must be omitted")
	}
}

func TestDeepResearch_InvalidLanguageInGetCached(t *testing.T) {
	svc, _ := setupMockDB(t)
	if _, err := svc.GetCachedResearch(context.Background(), uuid.New(), "T", "de"); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCacheTTLDefault(t *testing.T) {
	svc, _ := setupMockDB(t)
	if svc.cacheTTL != 0 {
		t.Fatal("expected zero TTL in test service (default fallback)")
	}
	svc.SetCacheTTL(2 * time.Hour)
	if svc.cacheTTL != 2*time.Hour {
		t.Error("SetCacheTTL must apply")
	}
}

func TestAIEntriesConversion(t *testing.T) {
	entries := aiFactsFromEntries([]FactBaseEntry{
		{FactType: FactTypeVersion, Entity: "App", Value: "1.0", SourceURL: "https://x.com", Confidence: 80},
	})
	if len(entries) != 1 || entries[0].Type != "version" || entries[0].Confidence != 80 || entries[0].Source != "https://x.com" {
		t.Errorf("fact conversion mismatch: %+v", entries)
	}

	srcs := aiSourcesFromEntries([]ResearchSource{
		{Title: "T", URL: "https://r.com", Domain: "r.com", ReliabilityScore: 95, ReliabilityLabel: "verified"},
	})
	if len(srcs) != 1 || srcs[0].ReliabilityLabel != "verified" {
		t.Errorf("source conversion mismatch: %+v", srcs)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSummaryFromReport(t *testing.T) {
	svc, _ := setupMockDB(t)
	report := &DeepResearchReport{
		ResearchJob: ResearchJob{Topic: "T", Language: "en"},
		Cached:      true,
		Briefing:    &ResearchBriefingDoc{Topic: "T", Summary: "S"},
		Facts:       []FactBaseEntry{{FactType: FactTypeNumber, Entity: "X", Value: "1"}},
	}
	summary := svc.summaryFromReport(report)
	if !summary.Cached || summary.Briefing == "" || len(summary.Facts) != 1 {
		t.Errorf("summary from report wrong: %+v", summary)
	}
}
