package publisher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

// qualityFixture builds deterministic, non-repetitive article paragraphs:
// each sentence varies 4 tokens (section, index, and two cycled words) so no
// two neighboring sentences share more than ~67% of their unique tokens —
// well below the 0.9 generic-content threshold. The text ends with a blank
// line so callers can chain sections/headings.
func qualityFixture(t *testing.T, section, sentences int) string {
	t.Helper()
	adjs := []string{"small", "fast", "smart", "modern"}
	nouns := []string{"teams", "startups", "agencies", "freelancers"}
	var b strings.Builder
	for i := 0; i < sentences; i++ {
		fmt.Fprintf(&b, "Section %d sentence %d explores how modern artificial intelligence tools improve everyday workflows for %s %s. ",
			section, i, adjs[i%len(adjs)], nouns[i%len(nouns)])
	}
	b.WriteString("\n\n")
	return b.String()
}

const qualityIntro = "Intro paragraph that introduces the topic and the reader value in a real sentence. " +
	"Another sentence to lengthen the introduction with more context about what the reader will learn. " +
	"A third sentence rounds out the introduction before the first heading.\n\n"

func TestCheckContentQuality_ShortArticleBlocked(t *testing.T) {
	// ~210 words, well under the 1000 minimum, with proper H2 structure so
	// the word count is the ONLY failing criterion.
	body := qualityIntro +
		"## First Section\n\n" + qualityFixture(t, 1, 4) +
		"## Second Section\n\n" + qualityFixture(t, 2, 4) +
		"## Third Section\n\n" + qualityFixture(t, 3, 4)

	res := CheckContentQuality(QualityGateInput{
		Title:       "How AI Tools Improve Team Workflows",
		Content:     body,
		Language:    "en",
		ContentType: "article",
	}, 1000, 3, 80)

	if res.Passed {
		t.Error("expected the short article to be blocked")
	}
	if res.WordCount >= 1000 {
		t.Errorf("fixture should be short, got %d words", res.WordCount)
	}
	if !hasIssue(res, "word_count") {
		t.Errorf("expected a word_count issue, got %+v", res.Issues)
	}
	if res.H2Count < 3 {
		t.Errorf("fixture should have 3 H2 sections, got %d", res.H2Count)
	}
	if len(res.Issues) != 1 {
		t.Errorf("word count must be the only failing criterion, got %+v", res.Issues)
	}
}

func TestCheckContentQuality_AdequateArticlePasses(t *testing.T) {
	body := qualityIntro +
		"## First Section\n\n" + qualityFixture(t, 1, 26) +
		"## Second Section\n\n" + qualityFixture(t, 2, 26) +
		"## Third Section\n\n" + qualityFixture(t, 3, 26)

	res := CheckContentQuality(QualityGateInput{
		Title:       "How AI Tools Improve Team Workflows",
		Content:     body,
		Language:    "en",
		ContentType: "article",
	}, 1000, 3, 80)

	if !res.Passed {
		t.Errorf("expected the adequate article to pass, got score %.2f issues %+v", res.Score, res.Issues)
	}
	if res.WordCount < 1000 {
		t.Errorf("fixture should exceed 1000 words, got %d", res.WordCount)
	}
	if res.Score != 100 {
		t.Errorf("expected a clean score of 100, got %.2f", res.Score)
	}
}

func TestCheckContentQuality_NewsTypeNeedsOnlyOneH2(t *testing.T) {
	body := qualityIntro +
		"## The Announcement\n\n" + qualityFixture(t, 1, 80)

	res := CheckContentQuality(QualityGateInput{
		Title:       "Company Announces New Product",
		Content:     body,
		Language:    "en",
		ContentType: "news",
	}, 1000, 3, 80)

	if !res.Passed {
		t.Errorf("news with a single H2 should pass the structure criterion, got %+v", res.Issues)
	}
}

func TestCheckContentQuality_ListWithoutExamplesBlocked(t *testing.T) {
	body := qualityIntro +
		"## First Item\n\n" + qualityFixture(t, 1, 26) +
		"## Second Item\n\n" + qualityFixture(t, 2, 26) +
		"## Third Item\n\n" + qualityFixture(t, 3, 26)

	res := CheckContentQuality(QualityGateInput{
		Title:       "Top 5 Tools for Teams",
		Content:     body,
		Language:    "en",
		ContentType: "top-list",
	}, 1000, 3, 80)

	if res.Passed {
		t.Error("expected a list article without H3/examples to be blocked")
	}
	if !hasIssue(res, "subsections") || !hasIssue(res, "examples") {
		t.Errorf("expected subsections+examples issues, got %+v", res.Issues)
	}
}

func TestCheckContentQuality_ListWithH3AndBulletsPasses(t *testing.T) {
	body := qualityIntro +
		"## First Item\n\n### Details\n\n- Feature one with real pricing and availability details\n\n" + qualityFixture(t, 1, 25) +
		"## Second Item\n\n### Details\n\n- Feature two with real pricing and availability details\n\n" + qualityFixture(t, 2, 25) +
		"## Third Item\n\n### Details\n\n- Feature three with real pricing and availability details\n\n" + qualityFixture(t, 3, 25)

	res := CheckContentQuality(QualityGateInput{
		Title:       "Top 5 Tools for Teams",
		Content:     body,
		Language:    "en",
		ContentType: "top-list",
	}, 1000, 3, 80)

	if !res.Passed {
		t.Errorf("expected the list with H3 and bullets to pass, got %+v", res.Issues)
	}
}

func TestCheckContentQuality_ResearchFactsRequireSources(t *testing.T) {
	body := qualityIntro +
		"## First Section\n\n" + qualityFixture(t, 1, 26) +
		"## Second Section\n\n" + qualityFixture(t, 2, 26) +
		"## Third Section\n\n" + qualityFixture(t, 3, 26)

	noSources := CheckContentQuality(QualityGateInput{
		Title:         "How AI Tools Improve Team Workflows",
		Content:       body,
		Language:      "en",
		ContentType:   "article",
		ResearchFacts: 5,
	}, 1000, 3, 80)

	if noSources.Passed || !hasIssue(noSources, "research") {
		t.Errorf("research facts without citations must block, got %+v", noSources.Issues)
	}

	withSources := CheckContentQuality(QualityGateInput{
		Title:         "How AI Tools Improve Team Workflows",
		Content:       body + "\n\n## Fontes\n\n- https://research.example.com/study",
		Language:      "en",
		ContentType:   "article",
		ResearchFacts: 5,
	}, 1000, 3, 80)

	if !withSources.Passed {
		t.Errorf("a Fontes section with a link must satisfy the research criterion, got %+v", withSources.Issues)
	}
}

func TestCheckContentQuality_GenericContentBlocked(t *testing.T) {
	// All sentences identical -> token Jaccard 1.0 between neighbors.
	body := qualityIntro +
		"## First Section\n\n" + strings.Repeat("The quick brown fox jumps over the lazy dog and chases the cat. ", 45) + "\n\n" +
		"## Second Section\n\n" + strings.Repeat("The quick brown fox jumps over the lazy dog and chases the cat. ", 45) + "\n\n" +
		"## Third Section\n\n" + strings.Repeat("The quick brown fox jumps over the lazy dog and chases the cat. ", 45) + "\n\n"

	res := CheckContentQuality(QualityGateInput{
		Title:       "How AI Tools Improve Team Workflows",
		Content:     body,
		Language:    "en",
		ContentType: "article",
	}, 1000, 3, 80)

	if res.Passed || !hasIssue(res, "generic") {
		t.Errorf("templated repetitive content must block, got %+v", res.Issues)
	}
}

func TestCheckContentQuality_Deterministic(t *testing.T) {
	body := qualityIntro +
		"## First Section\n\n" + qualityFixture(t, 1, 26) +
		"## Second Section\n\n" + qualityFixture(t, 2, 26) +
		"## Third Section\n\n" + qualityFixture(t, 3, 26)
	in := QualityGateInput{Title: "T", Content: body, Language: "en", ContentType: "article"}

	a := CheckContentQuality(in, 1000, 3, 80)
	b := CheckContentQuality(in, 1000, 3, 80)
	if a.Score != b.Score || len(a.Issues) != len(b.Issues) {
		t.Errorf("determinism violated: %+v vs %+v", a, b)
	}
}

func hasIssue(res *QualityGateResult, field string) bool {
	for _, i := range res.Issues {
		if i.Field == field {
			return true
		}
	}
	return false
}

// fakeQualityGate is a deterministic gate stub for funnel tests.
type fakeQualityGate struct {
	passed bool
	err    error
	calls  int
}

func (f *fakeQualityGate) CheckQuality(_ context.Context, in QualityGateInput) (*QualityGateResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	score := 50.0
	if f.passed {
		score = 95
	}
	return &QualityGateResult{Score: score, MinScore: 80, Passed: f.passed, WordCount: 1200, H2Count: 3}, nil
}

func TestPublishGeneratedArticle_QualityGateBlocksBeforeDB(t *testing.T) {
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = 80
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	fq := &fakeQualityGate{passed: false}
	svc.SetQualityGate(fq)

	_, err := svc.PublishGeneratedArticle(context.Background(), PublishGeneratedRequest{
		SiteID:   uuid.New(),
		Title:    "Artigo curto",
		Content:  "conteúdo raso",
		Language: "pt",
	})
	if !errors.Is(err, ErrQualityGateBlocked) {
		t.Errorf("expected ErrQualityGateBlocked before any DB access, got %v", err)
	}
	if fq.calls != 1 {
		t.Errorf("expected the quality gate to run exactly once, got %d", fq.calls)
	}
}

func TestPublishGeneratedArticle_QualityGateRunsBeforeSEOGate(t *testing.T) {
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = 80
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	fq := &fakeQualityGate{passed: true}
	svc.SetQualityGate(fq)
	svc.SetPublishGate(&fakeGate{score: 40})

	_, err := svc.PublishGeneratedArticle(context.Background(), PublishGeneratedRequest{
		SiteID:   uuid.New(),
		Title:    "Artigo bom",
		Content:  "conteúdo profundo",
		Language: "pt",
	})
	if !errors.Is(err, ErrSEOPublishBlocked) {
		t.Errorf("expected the SEO gate to block after a passing quality gate, got %v", err)
	}
	if fq.calls != 1 {
		t.Errorf("expected the quality gate to run before the SEO gate, got %d calls", fq.calls)
	}
}

func TestPublishGeneratedArticle_QualityGateFailsOpen(t *testing.T) {
	cfg := &config.Config{}
	cfg.SEO.MinPublishScore = 80
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	svc.SetQualityGate(&fakeQualityGate{err: errors.New("boom")})
	svc.SetPublishGate(&fakeGate{score: 40})

	_, err := svc.PublishGeneratedArticle(context.Background(), PublishGeneratedRequest{
		SiteID:   uuid.New(),
		Title:    "Artigo bom",
		Content:  "conteúdo profundo",
		Language: "pt",
	})
	if !errors.Is(err, ErrSEOPublishBlocked) {
		t.Errorf("expected fail-open to the SEO gate on quality gate error, got %v", err)
	}
}
