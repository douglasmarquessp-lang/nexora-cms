package seoengine

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/ai"
	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

// --- Sprint 6.8 gate-flow tests: FocusKeyword derivation, image → analyzer
// flow (Pexels and caller-provided), EEAT author boost, and the full
// integration article that must pass the 80/100 publish gate naturally.

func TestFocusKeyword(t *testing.T) {
	title := "5 Simple AI Tools That Can Save You Hours of Work Every Week"
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "two-word phrase present in body",
			body: "AI tools are everywhere today. We tested many AI tools for weeks.",
			want: "ai tools",
		},
		{
			name: "case-insensitive phrase match",
			body: "AI Tools changed our workflow. Simple tools save hours.",
			want: "ai tools",
		},
		{
			name: "singular token does not match plural phrase",
			body: "We use one AI tool for scheduling.",
			want: "ai",
		},
		{
			name: "phrase missing falls back to most repeated body term",
			body: "We work extra hours every week on repetitive work.",
			want: "work",
		},
		{
			name: "empty body falls back to deriveKeyword",
			body: "",
			want: "simple",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FocusKeyword(title, tt.body); got != tt.want {
				t.Fatalf("FocusKeyword() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFocusKeywordPortuguese(t *testing.T) {
	got := FocusKeyword("6 Formas Simples de Economizar Horas no Trabalho", "economizar horas no trabalho é simples")
	if got != "horas no trabalho" {
		t.Fatalf("FocusKeyword(pt) = %q, want %q", got, "horas no trabalho")
	}
}

type stubImageProvider struct{}

func (stubImageProvider) SearchImage(ctx context.Context, query string) (*PexelsImage, error) {
	return &PexelsImage{
		URL:             "https://images.pexels.com/photos/9/ai-tools.jpg",
		Alt:             "AI tools on a desk in an office",
		Photographer:    "Jane Doe",
		PhotographerURL: "https://pexels.com/@jane-doe",
		SourceURL:       "https://www.pexels.com/photo/9/",
	}, nil
}

func newGateTestService(defaultAuthor string) *Service {
	cfg := &config.Config{}
	cfg.SEO.DefaultAuthor = defaultAuthor
	return NewService(cfg, logger.New(&config.Config{}), nil, nil)
}

const sprint68Article = `# 5 AI Tools That Save Hours Every Week for Small Teams

We tested five AI tools in our own workflow in 2026, and they saved three to five hours per person every week. These helpers handle repetitive work, so your team can focus on what matters. This is a case study based on a real setup, not a sales pitch. Helpers that save hours are worth the time you invest in setup.

## Why Teams Waste Hours on Repetitive Work

According to a 2026 report, knowledge workers spend about 40% of their week on repetitive tasks. In our testing, simple automation removed most of that busywork. We measured the time before and after each tool.

## How AI Tools Save Three to Five Hours a Week

Each AI tool covers one task: drafting emails, summarizing meetings, writing code, building slides, and managing schedules. We ran a case study with a five-person team for eight weeks. The results were consistent. Surprisingly, they also improve quality.

### A Simple Setup for Small Teams

You do not need a big budget. Most AI tools have free tiers. Start with one tool, connect it to your calendar and database via API, and review the results every week. The setup took about 20 minutes in our testing.

AI automation means moving repetitive work to machines that follow a plan. An expert at OpenAI recommends small pilots before full adoption.

Last updated: January 2026. This post contains affiliate links.

## Sources

[OpenAI research](https://openai.com/research/automation) shows that simple prompts reduce manual work. Helpers rely on good data, and our report covers the details.

![AI tools on a desk in an office](https://images.pexels.com/photos/9/ai-tools.jpg)

- [Our AI chat widget](/ai-chat-widget)
- [The 2026 automation guide](/automation-guide)
`

var sprint68Meta = "Learn how five AI tools can save hours every week for small teams: automation, delegation, and real productivity gains tested by us in 2026"

func TestSprint68IntegrationArticlePassesGate(t *testing.T) {
	svc := newGateTestService("Sarah Chen")
	svc.SetQualityChecker(ai.NewQualityChecker())
	a := AnalyzeArticle(context.Background(), ArticleAnalysisInput{
		Title:           "5 AI Tools That Save Hours Every Week for Small Teams",
		MetaDescription: sprint68Meta,
		Keyword:         "ai tools",
		Language:        "en",
		AuthorName:      "Sarah Chen",
		Content:         sprint68Article,
	}, svc.qualityChecker)

	t.Logf("overall=%.2f title=%.2f meta=%.2f headings=%.2f keyword=%.2f readability=%.2f internal=%.2f external=%.2f eeat=%.2f images=%.2f (imgWithAlt=%d)",
		a.OverallScore, a.TitleScore, a.MetaScore, a.HeadingScore, a.KeywordScore,
		a.ReadabilityScore, a.InternalLinksScore, a.ExternalLinksScore, a.EEATScore, a.ImagesScore, a.ImagesWithAlt)

	if a.OverallScore < 80 {
		t.Fatalf("integration article scored %.2f, want >= 80", a.OverallScore)
	}
	// The dimensions that made failure possible score max/near-max so the
	// site's pin enforces only quality, never luck.
	for name, v := range map[string]float64{
		"title": a.TitleScore, "keywords": a.KeywordScore,
		"headings": a.HeadingScore, "internal": a.InternalLinksScore,
		"external": a.ExternalLinksScore, "images": a.ImagesScore,
	} {
		if v < 90 {
			t.Errorf("dimension %s = %.2f, want >= 90", name, v)
		}
	}
	if a.ImagesWithAlt < 1 {
		t.Errorf("ImagesWithAlt = %d, want >= 1 (the figure must be counted)", a.ImagesWithAlt)
	}
}

func TestSprint68PageLevelGate(t *testing.T) {
	svc := newGateTestService("Sarah Chen")
	svc.SetQualityChecker(ai.NewQualityChecker())
	in := publisher.PublishGateInput{
		SiteID:          uuid.New(),
		Title:           "5 AI Tools That Save Hours Every Week for Small Teams",
		Content:         sprint68Article,
		Language:        "en",
		MetaDescription: sprint68Meta,
		AuthorName:      "Sarah Chen",
	}
	score, err := svc.CheckPublishScore(context.Background(), in)
	if err != nil {
		t.Fatalf("CheckPublishScore: %v", err)
	}
	if score < 80 {
		t.Fatalf("gate score = %.2f, want >= 80 (no stored audit, no post id)", score)
	}

	// Same article with its keyword dropped must score lower — proves the
	// gate is sensitive to content quality, not a constant.
	bad := in
	bad.Content = strings.ReplaceAll(sprint68Article, "AI tools", "helpers")
	score2, err := svc.CheckPublishScore(context.Background(), bad)
	if err != nil {
		t.Fatalf("CheckPublishScore(bad): %v", err)
	}
	if score2 >= score {
		t.Fatalf("keyword-stripped article scored %.2f >= %.2f, gate must degrade", score2, score)
	}
	t.Logf("good=%.2f keyword-stripped=%.2f", score, score2)
}

func TestEnhanceEmbedsPexelsImageBeforeAnalysis(t *testing.T) {
	svc := newGateTestService("")
	svc.SetImageProvider(stubImageProvider{})

	base := strings.ReplaceAll(sprint68Article,
		"![AI tools on a desk in an office](https://images.pexels.com/photos/9/ai-tools.jpg)\n\n", "")

	out, err := svc.EnhanceBeforePublish(context.Background(), publisher.ContentEnhancerInput{
		SiteID:   uuid.New(),
		Title:    "5 AI Tools That Save Hours Every Week for Small Teams",
		Content:  base,
		Keyword:  "ai tools",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("EnhanceBeforePublish: %v", err)
	}
	if !strings.Contains(out.Content, "<figure>") ||
		!strings.Contains(out.Content, `src="https://images.pexels.com/photos/9/ai-tools.jpg"`) ||
		!strings.Contains(out.Content, `alt="AI tools on a desk in an office"`) {
		t.Fatalf("embedded figure missing from enhanced content:\n%s", out.Content)
	}

	svc.SetQualityChecker(ai.NewQualityChecker())
	withImg := AnalyzeArticle(context.Background(), ArticleAnalysisInput{
		Title: "5 AI Tools That Save Hours Every Week for Small Teams",
		Keyword: "ai tools",
		Language: "en",
		MetaDescription: sprint68Meta,
		Content: out.Content,
	}, svc.qualityChecker)
	withoutImg := AnalyzeArticle(context.Background(), ArticleAnalysisInput{
		Title: "5 AI Tools That Save Hours Every Week for Small Teams",
		Keyword: "ai tools",
		Language: "en",
		MetaDescription: sprint68Meta,
		Content: base,
	}, svc.qualityChecker)

	if withoutImg.ImagesScore != 30 {
		t.Fatalf("baseline ImagesScore = %.2f, want 30 (no image in body)", withoutImg.ImagesScore)
	}
	if withImg.ImagesScore != 100 {
		t.Fatalf("enhanced ImagesScore = %.2f, want 100 (figure embedded)", withImg.ImagesScore)
	}
	if withImg.OverallScore-withoutImg.OverallScore < 3 {
		t.Fatalf("enhanced overall (%.2f) must beat baseline (%.2f) by >= 3 pts", withImg.OverallScore, withoutImg.OverallScore)
	}
}

func TestEnhanceEmbedsCallerProvidedImage(t *testing.T) {
	svc := newGateTestService("")
	base := strings.ReplaceAll(sprint68Article,
		"![AI tools on a desk in an office](https://images.pexels.com/photos/9/ai-tools.jpg)\n\n", "")

	out, err := svc.EnhanceBeforePublish(context.Background(), publisher.ContentEnhancerInput{
		SiteID:           uuid.New(),
		Title:            "5 AI Tools That Save Hours Every Week for Small Teams",
		Content:          base,
		Keyword:          "ai tools",
		Language:         "en",
		FeaturedImageURL: "https://cdn.example.com/featured.jpg",
	})
	if err != nil {
		t.Fatalf("EnhanceBeforePublish: %v", err)
	}
	if !strings.Contains(out.Content, `<img src="https://cdn.example.com/featured.jpg"`) {
		t.Fatalf("caller-provided image not embedded in body:\n%s", out.Content)
	}
	if !strings.Contains(out.Content, `alt="tools save hours week small teams"`) {
		t.Fatalf("alt text missing off the caller-provided image:\n%s", out.Content)
	}
}

func TestEEATAuthorBoost(t *testing.T) {
	withAuthor := AnalyzeEEAT(ArticleAnalysisInput{
		Content:    sprint68Article,
		Keyword:    "ai tools",
		Language:   "en",
		AuthorName: "Sarah Chen",
	})
	withoutAuthor := AnalyzeEEAT(ArticleAnalysisInput{
		Content:  sprint68Article,
		Keyword:  "ai tools",
		Language: "en",
	})
	if withAuthor.Final-withoutAuthor.Final < 10 {
		t.Fatalf("author byline must boost EEAT: with=%.2f without=%.2f", withAuthor.Final, withoutAuthor.Final)
	}
}

func TestGateKeywordPassThrough(t *testing.T) {
	title := "5 Simple AI Tools That Can Save You Hours of Work Every Week"
	body := strings.ReplaceAll(sprint68Article,
		"![AI tools on a desk in an office](https://images.pexels.com/photos/9/ai-tools.jpg)\n\n", "")

	svc := newGateTestService("")
	svc.SetQualityChecker(ai.NewQualityChecker())

	// Derived (empty) and explicitly provided focus keyword must agree.
	derivedKw := FocusKeyword(title, body)
	derived := AnalyzeArticle(context.Background(), ArticleAnalysisInput{
		Title: title, MetaDescription: sprint68Meta, Content: body, Language: "en",
		Keyword: derivedKw,
	}, svc.qualityChecker)
	explicit := AnalyzeArticle(context.Background(), ArticleAnalysisInput{
		Title: title, MetaDescription: sprint68Meta, Content: body, Language: "en",
		Keyword: "ai tools",
	}, svc.qualityChecker)
	if derivedKw != "ai tools" {
		t.Fatalf("FocusKeyword = %q, want %q", derivedKw, "ai tools")
	}
	if derived.KeywordScore != explicit.KeywordScore {
		t.Fatalf("derived score %.2f != explicit score %.2f", derived.KeywordScore, explicit.KeywordScore)
	}
	if derived.OverallScore < explicit.OverallScore-0.01 || derived.OverallScore > explicit.OverallScore+0.01 {
		t.Fatalf("derived overall %.2f != explicit overall %.2f", derived.OverallScore, explicit.OverallScore)
	}
}

func TestCheckPublishScoreWithIssuesReportsImages(t *testing.T) {
	svc := newGateTestService("Sarah Chen")
	svc.SetQualityChecker(ai.NewQualityChecker())
	base := strings.ReplaceAll(sprint68Article,
		"![AI tools on a desk in an office](https://images.pexels.com/photos/9/ai-tools.jpg)\n\n", "")

	score, issues, err := svc.CheckPublishScoreWithIssues(context.Background(), publisher.PublishGateInput{
		SiteID: uuid.New(), Title: "5 AI Tools That Save Hours Every Week for Small Teams",
		Content: base, Language: "en", MetaDescription: sprint68Meta, AuthorName: "Sarah Chen",
	})
	if err != nil {
		t.Fatalf("CheckPublishScoreWithIssues: %v", err)
	}
	foundImages := false
	foundInternal := false
	for _, is := range issues {
		if strings.Contains(is, "images") {
			foundImages = true
		}
		if strings.Contains(is, "internal links") {
			foundInternal = true
		}
	}
	if !foundImages {
		t.Fatalf("expected images issue in list, got %v (score=%.2f)", issues, score)
	}
	if foundInternal {
		t.Fatalf("did not expect internal-links issue, got %v", issues)
	}
}