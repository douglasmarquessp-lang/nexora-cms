package freshness

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func mustTime(t *testing.T, s string) *time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		t.Fatalf("bad test time %q: %v", s, err)
	}
	return &parsed
}

func TestWindowStrategyNews(t *testing.T) {
	w := WindowStrategy(IntentNews)
	if w.PriorityDays != 1 || w.RecentDays != 7 || w.MaxDays != 30 || w.NeverOlderDays != 90 {
		t.Errorf("unexpected news window: %+v", w)
	}
}

func TestWindowStrategyEvergreenUnlimited(t *testing.T) {
	w := WindowStrategy(IntentEvergreen)
	if w.PriorityDays != 0 || w.MaxDays != 0 || w.NeverOlderDays != 0 {
		t.Errorf("expected unlimited evergreen window, got %+v", w)
	}
}

func TestWindowStrategyUpdateVersionFirst(t *testing.T) {
	w := WindowStrategy(IntentUpdate)
	if !w.VersionFirst {
		t.Error("expected update window to prioritize the latest version")
	}
}

func TestClassifyIntentNewsPT(t *testing.T) {
	ir, err := ClassifyIntent("", "Empresa anuncia hoje novo produto", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentNews {
		t.Errorf("expected news, got %s", ir.Intent)
	}
	if ir.Confidence <= 0 || ir.Confidence > 1 {
		t.Errorf("confidence out of range: %f", ir.Confidence)
	}
}

func TestClassifyIntentNewsEN(t *testing.T) {
	ir, err := ClassifyIntent("", "Tech company launches new chip today", "en")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentNews {
		t.Errorf("expected news, got %s", ir.Intent)
	}
}

func TestClassifyIntentEvergreenPT(t *testing.T) {
	ir, err := ClassifyIntent("", "Documentação de referência sobre PostgreSQL", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentEvergreen {
		t.Errorf("expected evergreen, got %s", ir.Intent)
	}
}

func TestClassifyIntentEvergreenFallback(t *testing.T) {
	ir, err := ClassifyIntent("Tópico neutro sem sinais", "", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentEvergreen {
		t.Errorf("expected evergreen fallback, got %s", ir.Intent)
	}
	if ir.Signals[len(ir.Signals)-1] != "fallback_evergreen" {
		t.Errorf("expected fallback signal, got %v", ir.Signals)
	}
}

func TestClassifyIntentUpdatePT(t *testing.T) {
	ir, err := ClassifyIntent("", "Changelog e novidades da nova versão do produto", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentUpdate {
		t.Errorf("expected update, got %s", ir.Intent)
	}
}

func TestClassifyIntentUpdateVersionBoost(t *testing.T) {
	ir, err := ClassifyIntent("", "O que mudou no GPT-6", "en")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentUpdate {
		t.Errorf("expected update via version token, got %s (signals %v)", ir.Intent, ir.Signals)
	}
}

func TestClassifyIntentReviewPT(t *testing.T) {
	ir, err := ClassifyIntent("", "Análise completa: testamos o produto", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentReview {
		t.Errorf("expected review, got %s", ir.Intent)
	}
}

func TestClassifyIntentTutorialEN(t *testing.T) {
	ir, err := ClassifyIntent("", "How to install a step by step guide", "en")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Intent != IntentTutorial {
		t.Errorf("expected tutorial, got %s", ir.Intent)
	}
}

func TestClassifyIntentInvalidLanguage(t *testing.T) {
	if _, err := ClassifyIntent("x", "y", "fr"); err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestClassifyIntentEmpty(t *testing.T) {
	if _, err := ClassifyIntent("", "", "pt"); err == nil {
		t.Error("expected error for empty topic+content")
	}
}

func TestComputeSourceFreshnessNewsFresh(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(0, 0, -2)
	upd := now.AddDate(0, 0, -1)
	br := ComputeSourceFreshness(now, &pub, &upd, IntentNews, "titulo", "https://docs.reuters.com/changelog", "")
	if !br.Usable {
		t.Error("expected fresh news source usable")
	}
	if br.Score < 85 {
		t.Errorf("expected high freshness for 2-day-old source, got %.2f", br.Score)
	}
	if br.SourcePriority != PriorityChangelog {
		t.Errorf("expected changelog priority, got %s", br.SourcePriority)
	}
}

func TestComputeSourceFreshnessNewsTooOld(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(0, 0, -45)
	br := ComputeSourceFreshness(now, &pub, nil, IntentNews, "t", "https://example.com/x", "")
	if br.Usable {
		t.Error("expected 45-day-old news source unusable (>30 day window)")
	}
}

func TestComputeSourceFreshnessNewsNeverOlder(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(0, 0, -120)
	br := ComputeSourceFreshness(now, &pub, nil, IntentNews, "t", "https://example.com", "")
	if br.Usable {
		t.Error("expected 120-day-old source unusable (never > 90 days)")
	}
	if br.Score > 60 {
		t.Errorf("expected badly penalized score, got %.2f", br.Score)
	}
}

func TestComputeSourceFreshnessEvergreenIgnoresAge(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(-2, 0, 0)
	br := ComputeSourceFreshness(now, &pub, nil, IntentEvergreen, "doc", "https://docs.example.com", "")
	if !br.Usable {
		t.Error("expected 2-year-old evergreen source usable (date not priority)")
	}
	if br.Score < 50 {
		t.Errorf("expected moderate evergreen score, got %.2f", br.Score)
	}
}

func TestSourcePriorityOfficial(t *testing.T) {
	p := SourcePriorityClassify("Official site", "https://openai.com/blog/gpt-6", "openai")
	if p != PriorityOfficial && p != PriorityBlog {
		t.Errorf("expected blog/official priority, got %s", p)
	}
}

func TestSourcePriorityDocs(t *testing.T) {
	p := SourcePriorityClassify("docs", "https://docs.python.org/3/", "")
	if p != PriorityDocs {
		t.Errorf("expected docs priority, got %s", p)
	}
}

func TestSourcePriorityAgency(t *testing.T) {
	p := SourcePriorityClassify("Reuters", "https://www.reuters.com/tech", "")
	if p != PriorityNewsAgency {
		t.Errorf("expected news agency priority, got %s", p)
	}
}

func TestSourcePrioritySpecialized(t *testing.T) {
	p := SourcePriorityClassify("article", "https://www.theverge.com/2026/1/1/abc", "")
	if p != PrioritySpecialized {
		t.Errorf("expected specialized priority, got %s", p)
	}
}

func TestSortByPriorityPutsObsoleteLast(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	a := ComputeSourceFreshness(now, nil, nil, IntentNews, "t", "https://docs.example.com/a", "")
	a.Obsolete = true
	b := ComputeSourceFreshness(now, nil, nil, IntentNews, "t", "https://example.com/b", "")
	sorted := SortSourcesByPriorityAndScore([]FreshnessBreakdown{a, b})
	if sorted[0].Obsolete {
		t.Error("expected obsolete source ranked last")
	}
}

func TestCheckObsoleteTrue(t *testing.T) {
	oc := CheckObsolete("A documentação de GPT-4 apresenta limites de contexto.", EntityVersion{Entity: "GPT", Current: "6"})
	if !oc.Obsolete {
		t.Errorf("expected GPT-4 mention flagged obsolete, got %+v", oc)
	}
	if oc.MentionedVersion != "4" {
		t.Errorf("expected mentioned 4, got %s", oc.MentionedVersion)
	}
}

func TestCheckObsoleteFalse(t *testing.T) {
	oc := CheckObsolete("GPT-6 está disponível hoje", EntityVersion{Entity: "GPT", Current: "6"})
	if oc.Obsolete {
		t.Error("expected current version NOT obsolete")
	}
}

func TestCheckObsoleteNoMatch(t *testing.T) {
	oc := CheckObsolete("apenas conteudo generico aqui", EntityVersion{Entity: "GPT", Current: "6"})
	if oc.Obsolete {
		t.Error("no mention must not be obsolete")
	}
}

func TestDetectObsoleteSources(t *testing.T) {
	entities := []EntityVersion{{Entity: "GPT", Current: "6"}, {Entity: "Gemini", Current: "2.5"}}
	oc := DetectObsoleteSources("O GPT-4 Turbo é o modelo mais recente.", entities)
	if len(oc) != 1 {
		t.Errorf("expected exactly 1 obsolete flag, got %d", len(oc))
	}
}

func TestNextVersion(t *testing.T) {
	if NextVersion("v1.2") != "v1.3" {
		t.Errorf("unexpected bump: %s", NextVersion("v1.2"))
	}
	if NextVersion("") != "v1" {
		t.Errorf("unexpected default: %s", NextVersion(""))
	}
}

func TestDiffVersionsDetectsChange(t *testing.T) {
	oldT := "Preço: US$ 20 por mês. Limite: 500 requests. Benchmark: 70."
	newT := "Preço: US$ 25 por mês. Limite: 1000 requests. Benchmark: 81."
	diffs := DiffVersions(oldT, newT)
	changed := 0
	for _, d := range diffs {
		if d.Changed {
			changed++
		}
	}
	if changed == 0 {
		t.Error("expected at least one facet changed")
	}
}

func TestDiffVersionsUnchanged(t *testing.T) {
	txt := "Preço: US$ 20/mês. Limite: 500."
	diffs := DiffVersions(txt, txt)
	for _, d := range diffs {
		if d.Before != d.After {
			t.Errorf("facet %s changed unexpectedly", d.Facet)
		}
	}
}

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint("GPT-6 lançamento", "OpenAI lançou o GPT-6 hoje.")
	b := Fingerprint("GPT-6 lançamento", "OpenAI lançou o GPT-6 hoje.")
	if a != b || a == "" {
		t.Errorf("expected stable fingerprint, got %s vs %s", a, b)
	}
}

func TestCheckDuplicateSameDay(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	topic := "Empresa anuncia novo modelo"
	existing := []Candidate{{
		PublicationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Topic:         "Empresa anuncia novo modelo de IA",
		PublishedAt:   &now,
	}}
	dc := CheckDuplicate(topic, "Conteúdo da notícia sobre o novo modelo anunciado pela empresa.", "pt", now, existing)
	if !dc.Duplicate || !dc.SameDay {
		t.Errorf("expected duplicate same-day match, got %+v", dc)
	}
}

func TestCheckDuplicateNoMatch(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)
	existing := []Candidate{{
		PublicationID: uuid.New(), Topic: "Outono europeu 2026", PublishedAt: &yesterday,
	}}
	dc := CheckDuplicate("GPT-6 lançamento", "conteúdo completo sobre o lançamento", "pt", now, existing)
	if dc.Duplicate {
		t.Errorf("expected no duplicate, got %+v", dc)
	}
}

func TestReEvaluateArticleNeedsUpdate(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(0, 0, -60)
	dec := ReEvaluateArticle(pub, IntentNews, now, false)
	if !dec.NeedsUpdate {
		t.Error("expected news article outside 30-day window marked needs update")
	}
	if dec.Reason != "outside_temporal_window" {
		t.Errorf("unexpected reason: %s", dec.Reason)
	}
}

func TestReEvaluateArticleNewerSource(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(0, 0, -3)
	dec := ReEvaluateArticle(pub, IntentNews, now, true)
	if !dec.NeedsUpdate || dec.Reason != "newer_source_found" {
		t.Errorf("expected newer_source_found, got %+v", dec)
	}
}

func TestReEvaluateArticleFresh(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pub := now.AddDate(0, 0, -1)
	dec := ReEvaluateArticle(pub, IntentNews, now, false)
	if dec.NeedsUpdate {
		t.Error("expected fresh news article NOT marked for update")
	}
}

func TestDailySweepOnceGuard(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if !DailySweepOnce(nil, now) {
		t.Error("expected first sweep allowed")
	}
	if DailySweepOnce(&now, now) {
		t.Error("expected same-day sweep blocked")
	}
	next := now.AddDate(0, 0, 1)
	if !DailySweepOnce(&now, next) {
		t.Error("expected next-day sweep allowed")
	}
}

func TestLanguageStrategyPT(t *testing.T) {
	ls := LanguageStrategyFor("pt")
	if len(ls.PrimaryRegions) != 1 || ls.PrimaryRegions[0] != "Brasil" {
		t.Errorf("unexpected pt regions: %v", ls.PrimaryRegions)
	}
	if !ls.BlockLiteralTranslation {
		t.Error("expected PT to block literal translation when english sources exist")
	}
}

func TestLanguageStrategyEN(t *testing.T) {
	ls := LanguageStrategyFor("en")
	if len(ls.PrimaryRegions) != 2 {
		t.Errorf("expected EUA+Canadá primary, got %v", ls.PrimaryRegions)
	}
}

func TestSourceMatchesRegion(t *testing.T) {
	if !SourceMatchesRegion("globo.com.br", "pt") {
		t.Error("expected .br PT domain accepted")
	}
	if SourceMatchesRegion("example.co.uk", "pt") {
		t.Error("expected .co.uk rejected for pt")
	}
	if !SourceMatchesRegion("example.co.uk", "en") {
		t.Error("expected .co.uk accepted for en")
	}
}

func TestDeriveMainEntity(t *testing.T) {
	if e := DeriveMainEntity("OpenAI GPT-6"); e != "openai" {
		t.Errorf("expected openai, got %q", e)
	}
}

func TestSourcePriorityClassifyTarget(t *testing.T) {
	p := SourcePriorityClassify("", "https://gemini.google.com/app", "gemini")
	if p != PriorityOfficial {
		t.Errorf("expected official for brand domain, got %s", p)
	}
}

func TestCheckObsoleteVersionBigger(t *testing.T) {
	oc := CheckObsolete("O Gemini 2.5 superou benchmarks", EntityVersion{Entity: "Gemini", Current: "2.0"})
	if oc.Obsolete {
		t.Error("expected NEWER version NOT obsolete")
	}
}

func TestCompareEntities(t *testing.T) {
	if compareEntities("2.5", "6") >= 0 {
		t.Error("expected 2.5 < 6")
	}
	if compareEntities("6", "6") != 0 {
		t.Error("expected equal")
	}
	if compareEntities("2.0", "2.5") >= 0 {
		t.Error("expected 2.0 < 2.5")
	}
}