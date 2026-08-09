package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunSEO_KeepsArticleAndAddsSources(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	article := "# Mock Title\n\nBase article body written by the writer stage for testing purposes.\n\n## section\nSome content here."
	result, err := pe.ExecuteStage(ctx, StageSEOGen, PipelineInput{
		Content:  article,
		Keywords: []string{"mock"},
		Language: "en",
		GroundingMetadata: &GroundingMetadata{
			Sources: []GroundingSource{
				{URI: "https://news.example.gov/entry", Title: "Official press release", IsVerified: true, RetrievedAt: now},
				{URI: "https://low.example/entry", Title: "Low reliability blog", IsVerified: false, RetrievedAt: now},
			},
		},
	})
	if err != nil {
		t.Fatalf("runSEO failed: %v", err)
	}
	if !strings.Contains(result.Content, "Mock Title") {
		t.Error("SEO stage must keep the article in Content")
	}
	if !strings.Contains(result.Content, "[Official press release](https://news.example.gov/entry)") {
		t.Error("expected verified external source appended as markdown link")
	}
	if strings.Contains(result.Content, "low.example") {
		t.Error("low-reliability source must not be appended")
	}
	if !strings.Contains(result.Analysis, "Mock") {
		t.Error("SEO recommendations must go to Analysis, not Content")
	}
}

// A draft with no H2 subheadings gets an honest "Introduction" heading before
// the sources section — the published page never stays heading-less.
func TestPipelineSEO_AddsIntroHeadingWhenNoH2(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageSEOGen, PipelineInput{
		Content:  "Only a flat paragraph without any subheading.",
		Keywords: []string{"x"},
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("runSEO failed: %v", err)
	}
	if !strings.Contains(result.Content, "## Introdução") {
		t.Errorf("expected an Introduction H2 in PT draft, got:\n%s", result.Content)
	}
}

func TestPipelineSEO_KeepsExistingH2(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageSEOGen, PipelineInput{
		Content:  "# T\n\n## Existing section\nbody.\n## Second\nmore.",
		Keywords: []string{"x"},
		Language: "en",
	})
	if err != nil {
		t.Fatalf("runSEO failed: %v", err)
	}
	if strings.Contains(result.Content, "## Introduction") {
		t.Error("must not add an Introduction when the article already has H2s")
	}
}

func TestSourcesSection_FiltersAndCaps(t *testing.T) {
	gm := &GroundingMetadata{Sources: []GroundingSource{
		{URI: "https://reuters.com/story/1", Title: ""}, // title falls back to domain
		{URI: "https://rg.example/blog-post", Title: "low"},
		{URI: "https://www.gov.br/noticia", Title: "gov"},
		{URI: "https://example.ieee.org/paper", Title: "ieee"},
		{URI: "https://university.edu/research", Title: "university"},
		{URI: "https://acm.org/study", Title: "acm"},
		{URI: "https://en.wikipedia.org/repo", Title: "wiki"},
		{URI: "https://reuters.com/story/one", Title: "duplicate"},
		{URI: "", Title: "empty"},
	}}
	out := sourcesSection(gm, "en")
	lines := strings.Split(out, "\n")
	// heading + up to 5 links
	linkCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "- [") {
			linkCount++
		}
	}
	if linkCount != 5 {
		t.Errorf("expected 5 links max, got %d:\n%s", linkCount, out)
	}
	if strings.Contains(out, "rg.example") {
		t.Error("low reliability domain must be filtered out")
	}
	if !strings.Contains(out, "[reuters.com]") {
		t.Error("empty title should fall back to the domain")
	}
	if strings.Contains(out, "duplicate") {
		t.Error("duplicate URI must be deduplicated")
	}
}

func TestSourcesSection_Empty(t *testing.T) {
	if s := sourcesSection(nil, "pt"); s != "" {
		t.Errorf("nil metadata must return empty, got %q", s)
	}
	if s := sourcesSection(&GroundingMetadata{}, "pt"); s != "" {
		t.Errorf("empty metadata must return empty, got %q", s)
	}
}

func TestEnsureIntroHeading(t *testing.T) {
	if got := ensureIntroHeading("some text", "en"); !strings.Contains(got, "## Introduction") {
		t.Errorf("expected EN Introduction heading, got %q", got)
	}
	if got := ensureIntroHeading("some text", "pt"); !strings.Contains(got, "## Introdução") {
		t.Errorf("expected PT Introdução heading, got %q", got)
	}
	if got := ensureIntroHeading("## Real section\ntext", "pt"); strings.Contains(got, "## Introdução") {
		t.Error("must not introduce a heading when one already exists")
	}
	if got := ensureIntroHeading("", "pt"); got != "" {
		t.Errorf("empty content must stay empty, got %q", got)
	}
}
