package publisher

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/sitedomain"
)

// aiWorkSimpleID mirrors the sitelang pin used in production (English-only).
const aiWorkSimpleID = "a64d7d72-b97f-4f31-96fd-8aeb15f6184c"

// fakeSiteResolver is a deterministic sitedomain.Resolver for tests.
type fakeSiteResolver struct {
	sc  sitedomain.SiteContext
	err error
}

func (f *fakeSiteResolver) Resolve(ctx context.Context, siteID uuid.UUID) (sitedomain.SiteContext, error) {
	return f.sc, f.err
}

// captureArgs implements pgxmock.Argument to collect the string arguments of
// an executed statement (the UPDATE publications arg order is random because
// the updates map iterates non-deterministically).
type captureArgs struct {
	got []string
}

func (c *captureArgs) Match(v interface{}) bool {
	if s, ok := v.(string); ok {
		c.got = append(c.got, s)
	}
	return true
}

func containsAny(haystack []string, needles ...string) bool {
	for _, n := range needles {
		for _, h := range haystack {
			if h == n {
				return true
			}
		}
	}
	return false
}

func siteContextSvc(t *testing.T, sc sitedomain.SiteContext) (*Service, pgxmock.PgxPoolIface) {
	t.Helper()
	svc, mock := setupMockDB(t)
	svc.SetSiteResolver(&fakeSiteResolver{sc: sc})
	return svc, mock
}

func expectInsertPublication(mock pgxmock.PgxPoolIface) {
	args := make([]interface{}, 35)
	for i := range args {
		args[i] = pgxmock.AnyArg()
	}
	mock.ExpectExec(`INSERT INTO publications`).
		WithArgs(args...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	histArgs := make([]interface{}, 12)
	for i := range histArgs {
		histArgs[i] = pgxmock.AnyArg()
	}
	mock.ExpectExec(`INSERT INTO publication_history`).
		WithArgs(histArgs...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

// --- PublishArticle: domain + primary language per site ---

func TestPublishArticle_EN_PrimaryLanguage(t *testing.T) {
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "en",
	})

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(siteID, "test-article").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)

	resp, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:    "Test Article",
		Language: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Publication.URL != "https://aiworksimple.com/test-article" {
		t.Errorf("url = %q, want root without language prefix", resp.Publication.URL)
	}
	if resp.Publication.CanonicalURL != "https://aiworksimple.com/test-article" {
		t.Errorf("canonical = %q, want root for primary language", resp.Publication.CanonicalURL)
	}
	if resp.Publication.Language != "en" {
		t.Errorf("language = %q, want %q", resp.Publication.Language, "en")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPublishArticle_PT_PrimaryLanguage(t *testing.T) {
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://dominio-pt.com",
		PrimaryLanguage: "pt",
	})

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(siteID, "teste").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)

	resp, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:    "Teste",
		Language: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Publication.URL != "https://dominio-pt.com/teste" {
		t.Errorf("url = %q, want root", resp.Publication.URL)
	}
	if resp.Publication.CanonicalURL != "https://dominio-pt.com/teste" {
		t.Errorf("canonical = %q, want root for primary language", resp.Publication.CanonicalURL)
	}
	if resp.Publication.Language != "pt" {
		t.Errorf("language = %q, want %q", resp.Publication.Language, "pt")
	}
}

func TestPublishArticle_SecondaryLanguage(t *testing.T) {
	// EN site, explicit pt request (site is NOT pinned) → /pt/ prefix.
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "en",
	})

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(siteID, "test-article").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)

	resp, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:    "Test Article",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Publication.URL != "https://aiworksimple.com/pt/test-article" {
		t.Errorf("url = %q, want /pt/ prefix for secondary language", resp.Publication.URL)
	}
	if resp.Publication.CanonicalURL != "https://aiworksimple.com/pt/test-article" {
		t.Errorf("canonical = %q, want /pt/ prefix for secondary language", resp.Publication.CanonicalURL)
	}
	if resp.Publication.Language != "pt" {
		t.Errorf("language = %q, want %q", resp.Publication.Language, "pt")
	}
}

func TestPublishArticle_PinWinsOverSiteLanguage(t *testing.T) {
	// AIWorkSimple pinned to "en" even when the site locale says pt — the
	// sitelang override keeps its precedence.
	pinID, err := uuid.Parse(aiWorkSimpleID)
	if err != nil {
		t.Fatalf("invalid pin id: %v", err)
	}
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "pt",
	})

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(pinID, "test-article").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)

	resp, err := svc.PublishArticle(context.Background(), pinID, uuid.New(), PublishRequest{
		Title:    "Test Article",
		Language: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Publication.Language != "en" {
		t.Errorf("language = %q, want pinned %q", resp.Publication.Language, "en")
	}
	if resp.Publication.URL != "https://aiworksimple.com/test-article" {
		t.Errorf("url = %q, want root in pinned language", resp.Publication.URL)
	}
}

func TestPublishArticle_MultilingualURLs_PrimaryEN(t *testing.T) {
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "en",
	})

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(siteID, "test-article").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)

	resp, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:        "Test Article",
		Language:     "en",
		Translations: map[string]interface{}{"en": "test-article", "pt": "teste-artigo"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ml := resp.Publication.MultilingualURLs
	if ml["en"] != "https://aiworksimple.com/test-article" {
		t.Errorf("en url = %v, want root", ml["en"])
	}
	if ml["pt"] != "https://aiworksimple.com/pt/test-article" {
		t.Errorf("pt url = %v, want /pt/ prefix", ml["pt"])
	}
}

func TestPublishArticle_NoResolver_ExplicitError(t *testing.T) {
	// No resolver registered → publishing fails explicitly; the service never
	// falls back to a placeholder domain.
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	siteID := uuid.New()
	_, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:    "Teste",
		Language: "",
	})
	if !errors.Is(err, ErrDomainUnresolved) {
		t.Errorf("expected ErrDomainUnresolved (no resolver), got %v", err)
	}

	// URL preview path without resolver: explicit error, never example.com.
	_, err = svc.GenerateSlugURL(context.Background(), siteID, "teste")
	if !errors.Is(err, ErrDomainUnresolved) {
		t.Errorf("expected ErrDomainUnresolved for slug URL, got %v", err)
	}
}

func TestPublishArticle_NoVerifiedDomain_ExplicitError(t *testing.T) {
	// Resolver registered but the site has no verified domain → the service
	// must fail explicitly instead of emitting a placeholder domain.
	svc, _ := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "",
		PrimaryLanguage: "pt",
	})
	siteID := uuid.New()

	_, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:    "Teste",
		Language: "",
	})
	if !errors.Is(err, ErrNoVerifiedDomain) {
		t.Errorf("expected ErrNoVerifiedDomain, got %v", err)
	}

	_, err = svc.GenerateSlugURL(context.Background(), siteID, "teste")
	if !errors.Is(err, ErrNoVerifiedDomain) {
		t.Errorf("expected ErrNoVerifiedDomain for slug URL, got %v", err)
	}
}

func TestPublishArticle_ResolverError_ExplicitError(t *testing.T) {
	// Resolver failure (degraded DB) → publish fails explicitly.
	svc, _ := siteContextSvc(t, sitedomain.SiteContext{})
	svc.SetSiteResolver(&fakeSiteResolver{err: errors.New("db down")})
	siteID := uuid.New()

	_, err := svc.PublishArticle(context.Background(), siteID, uuid.New(), PublishRequest{
		Title:    "Teste",
		Language: "",
	})
	if !errors.Is(err, ErrDomainUnresolved) {
		t.Errorf("expected ErrDomainUnresolved (resolver error), got %v", err)
	}
}

func TestPublishArticle_AIWorkSimple_OfficialDomain(t *testing.T) {
	// AIWorkSimple resolves to its official domain; the placeholder domain is
	// never used anywhere in the published URLs.
	pinID, err := uuid.Parse(aiWorkSimpleID)
	if err != nil {
		t.Fatalf("invalid pin id: %v", err)
	}
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://www.aiworksimple.com",
		PrimaryLanguage: "en",
	})

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(pinID, "test-article").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)

	resp, err := svc.PublishArticle(context.Background(), pinID, uuid.New(), PublishRequest{
		Title:    "Test Article",
		Language: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Publication.URL != "https://www.aiworksimple.com/test-article" {
		t.Errorf("url = %q, want official domain root", resp.Publication.URL)
	}
	if resp.Publication.CanonicalURL != "https://www.aiworksimple.com/test-article" {
		t.Errorf("canonical = %q, want official domain root", resp.Publication.CanonicalURL)
	}
	if resp.Publication.Language != "en" {
		t.Errorf("language = %q, want pinned %q", resp.Publication.Language, "en")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- PublishGeneratedArticle ---

func TestPublishGeneratedArticle_EN_UsesResolvedDomain(t *testing.T) {
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = 80
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	fg := &fakeGate{score: 100}
	svc.SetPublishGate(fg)
	svc.SetSiteResolver(&fakeSiteResolver{sc: sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "en",
	}})

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(siteID, "test-article").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	expectInsertPublication(mock)
	auditArgs := make([]interface{}, 7)
	for i := range auditArgs {
		auditArgs[i] = pgxmock.AnyArg()
	}
	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(auditArgs...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	pub, err := svc.PublishGeneratedArticle(context.Background(), PublishGeneratedRequest{
		SiteID:   siteID,
		Title:    "Test Article",
		Content:  "Content with enough words to pass the gate",
		Language: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub.URL != "https://aiworksimple.com/test-article" {
		t.Errorf("url = %q, want resolved domain root", pub.URL)
	}
	if pub.Language != "en" {
		t.Errorf("language = %q, want %q", pub.Language, "en")
	}
	if fg.got == nil {
		t.Fatal("expected gate consultation")
	}
	if fg.got.Language != "en" {
		t.Errorf("gate language = %q, want %q", fg.got.Language, "en")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- UpdatePublication: rename slug regenerates URLs from the site ---

func TestUpdatePublication_RenameSlug_ResolvedDomain(t *testing.T) {
	svc, mock := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "en",
	})

	siteID := uuid.New()
	pubID := uuid.New()
	nowTime := now()
	newSlug := "new-slug"

	// Existing publication: pt language, old URL, canonical contains old slug
	// → the service must regenerate both from the resolved site context.
	mock.ExpectQuery(`SELECT .+ FROM publications WHERE`).
		WithArgs(pubID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "post_id", "title", "content", "excerpt", "slug", "url",
			"canonical_url", "language", "translations", "multilingual_urls",
			"status", "visibility", "author_id", "published_by", "published_at", "unpublished_at",
			"scheduled_at", "is_featured", "meta_title", "meta_description", "og_image",
			"featured_image_url", "tags", "categories", "word_count", "reading_time", "revision",
			"checksum", "source", "metadata", "created_by", "created_at", "updated_at",
		}).AddRow(pubID, siteID, nil, "Test", "content", "", "test-slug", "https://example.com/test-slug",
			"https://example.com/test-slug", "pt", "{}", "{}",
			"published", "public", nil, nil, nil, nil,
			nil, false, "", "", "",
			"", []string{}, []string{}, 0, 0, 1,
			"", "manual", "{}", nil, nowTime, nowTime))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(siteID, newSlug, pubID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	// Capture the UPDATE args: url + canonical must use the resolved domain
	// and the site's primary language (en) → secondary pt keeps the /pt/ prefix.
	// The same matcher instance is registered on every position because the
	// updates map iterates non-deterministically.
	captured := &captureArgs{}
	mock.ExpectExec(`UPDATE publications SET`).
		WithArgs(captured, captured, captured, captured, captured, captured).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	histArgs := make([]interface{}, 12)
	for i := range histArgs {
		histArgs[i] = pgxmock.AnyArg()
	}
	mock.ExpectExec(`INSERT INTO publication_history`).
		WithArgs(histArgs...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Final read returns the (mock) persisted row.
	mock.ExpectQuery(`SELECT .+ FROM publications WHERE`).
		WithArgs(pubID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "post_id", "title", "content", "excerpt", "slug", "url",
			"canonical_url", "language", "translations", "multilingual_urls",
			"status", "visibility", "author_id", "published_by", "published_at", "unpublished_at",
			"scheduled_at", "is_featured", "meta_title", "meta_description", "og_image",
			"featured_image_url", "tags", "categories", "word_count", "reading_time", "revision",
			"checksum", "source", "metadata", "created_by", "created_at", "updated_at",
		}).AddRow(pubID, siteID, nil, "Test", "content", "", newSlug, "https://aiworksimple.com/pt/new-slug",
			"https://aiworksimple.com/pt/new-slug", "pt", "{}", "{}",
			"published", "public", nil, nil, nil, nil,
			nil, false, "", "", "",
			"", []string{}, []string{}, 0, 0, 2,
			"", "manual", "{}", nil, nowTime, nowTime))

	req := UpdatePublicationRequest{Slug: &newSlug}
	pub, err := svc.UpdatePublication(context.Background(), siteID, uuid.New(), pubID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsAny(captured.got,
		"https://aiworksimple.com/pt/new-slug") {
		t.Errorf("UPDATE args missing resolved URL, got %v", captured.got)
	}
	if !containsAny(captured.got,
		"https://aiworksimple.com/pt/new-slug") {
		t.Errorf("UPDATE args missing resolved canonical, got %v", captured.got)
	}
	if pub.Slug != newSlug {
		t.Errorf("slug = %q, want %q", pub.Slug, newSlug)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- GenerateSlugURL ---

func TestGenerateSlugURL_ResolvedDomain(t *testing.T) {
	svc, _ := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://aiworksimple.com",
		PrimaryLanguage: "en",
	})

	url, err := svc.GenerateSlugURL(context.Background(), uuid.New(), "test-article")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://aiworksimple.com/test-article" {
		t.Errorf("url = %q, want %q", url, "https://aiworksimple.com/test-article")
	}
}

func TestGenerateSlugURL_PTSite(t *testing.T) {
	svc, _ := siteContextSvc(t, sitedomain.SiteContext{
		Domain:          "https://dominio-pt.com",
		PrimaryLanguage: "pt",
	})

	url, err := svc.GenerateSlugURL(context.Background(), uuid.New(), "teste")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://dominio-pt.com/teste" {
		t.Errorf("url = %q, want %q", url, "https://dominio-pt.com/teste")
	}
}

// --- buildMultilingualURLs ---

func TestBuildMultilingualURLs_PrimaryEN(t *testing.T) {
	got := buildMultilingualURLs(
		map[string]interface{}{"en": "x", "pt": "y"},
		"article", "https://aiworksimple.com", "en",
	)
	if got["en"] != "https://aiworksimple.com/article" {
		t.Errorf("en = %v, want root", got["en"])
	}
	if got["pt"] != "https://aiworksimple.com/pt/article" {
		t.Errorf("pt = %v, want /pt/ prefix", got["pt"])
	}
}

func TestBuildMultilingualURLs_PrimaryPT(t *testing.T) {
	got := buildMultilingualURLs(
		map[string]interface{}{"en": "x", "pt": "y"},
		"artigo", "https://dominio-pt.com", "pt",
	)
	if got["pt"] != "https://dominio-pt.com/artigo" {
		t.Errorf("pt = %v, want root", got["pt"])
	}
	if got["en"] != "https://dominio-pt.com/en/artigo" {
		t.Errorf("en = %v, want /en/ prefix", got["en"])
	}
}

func TestBuildMultilingualURLs_EmptyPrimaryFallsBackToPT(t *testing.T) {
	got := buildMultilingualURLs(
		map[string]interface{}{"pt": "x", "en": "y"},
		"artigo", "https://dominio-pt.com", "",
	)
	if got["pt"] != "https://dominio-pt.com/artigo" {
		t.Errorf("pt = %v, want root (legacy behavior)", got["pt"])
	}
	if got["en"] != "https://dominio-pt.com/en/artigo" {
		t.Errorf("en = %v, want /en/ prefix", got["en"])
	}
}

func TestBuildMultilingualURLs_NilTranslations(t *testing.T) {
	got := buildMultilingualURLs(nil, "artigo", "https://dominio-pt.com", "pt")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}
