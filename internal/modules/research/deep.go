package research

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/ai"
)

// researchTopicHash derives a stable 16-char key for a topic (case-folded).
func researchTopicHash(topic string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(topic))))
	return hex.EncodeToString(sum[:])[:16]
}

// DeepResearch performs the full research workflow for a topic:
// cache lookup → multi-source search (grounding) → reliability ranking →
// fact base extraction → structured briefing → persistence → cache write.
// Results are cached for cfg.Research.CacheTTL (default 24h).
func (s *Service) DeepResearch(ctx context.Context, siteID uuid.UUID, topic, language string) (*DeepResearchReport, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, ErrTopicRequired
	}
	lang := strings.ToLower(language)
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	hash := researchTopicHash(topic)

	// 1. Cache lookup (24h TTL by default).
	if cached, err := s.getCached(ctx, siteID, hash, lang); err == nil && cached != nil {
		report := s.reportFromCache(cached)
		s.fireEvent(ctx, EventResearchCached, map[string]interface{}{
			"topic":   topic,
			"site_id": siteID.String(),
			"cached":  true,
		}, siteID)
		return report, nil
	}

	// 2. Multi-source search.
	result, err := s.ExecuteGroundedResearch(ctx, topic, lang)
	if err != nil {
		return nil, fmt.Errorf("deep research: search: %w", err)
	}

	// 3. Persist the research job.
	job, err := s.CreateJob(ctx, siteID, uuid.Nil, CreateResearchJobRequest{Topic: topic, Language: lang})
	if err != nil {
		return nil, fmt.Errorf("deep research: create job: %w", err)
	}

	// 4. Convert grounding sources, rank by reliability, persist.
	sources := s.SourcesFromGrounding(job.ID, result.GroundingMetadata)
	sources = s.decorateSources(sources)
	sources = rankSources(sources)
	var persisted []ResearchSource
	for _, src := range sources {
		if src.URL == "" {
			continue
		}
		saved, err := s.AddSource(ctx, job.ID, src)
		if err != nil {
			return nil, fmt.Errorf("deep research: add source: %w", err)
		}
		persisted = append(persisted, *saved)
	}
	sources = persisted

	// 5. Fact base extraction (AI-assisted, deterministic fallback).
	var srcTexts []SourceText
	for _, src := range sources {
		srcTexts = append(srcTexts, SourceText{Title: src.Title, Snippet: src.Summary, URL: src.URL})
	}
	facts := s.ExtractFactBaseAI(ctx, topic, srcTexts)
	for i := range facts {
		facts[i].SiteID = siteID
		facts[i].ResearchJobID = job.ID
	}
	if err := s.saveFactBase(ctx, facts); err != nil {
		return nil, fmt.Errorf("deep research: save fact base: %w", err)
	}

	// 6. Structured briefing (AI-assisted, deterministic fallback).
	briefing := s.BuildBriefingAI(ctx, topic, lang, sources, facts)
	if err := s.persistBriefing(ctx, job.ID, briefing); err != nil {
		return nil, fmt.Errorf("deep research: save briefing: %w", err)
	}

	// 7. Cache the result for reuse (translation engine, future articles).
	if err := s.saveCache(ctx, siteID, topic, hash, lang, briefing, facts, sources); err != nil {
		s.log.Error("deep research: failed to write cache", "topic", topic, "error", err)
	}

	// 8. Complete the job.
	_ = s.CompleteJob(ctx, siteID, job.ID)

	return &DeepResearchReport{
		ResearchJob: *job,
		Briefing:    &briefing,
		Facts:       facts,
		Sources:     sources,
		Cached:      false,
	}, nil
}

// GetCachedResearch returns the cached research for a topic when still fresh;
// ErrCacheEntryNotFound when missing or expired. It never triggers research.
func (s *Service) GetCachedResearch(ctx context.Context, siteID uuid.UUID, topic, language string) (*CachedResearch, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, ErrTopicRequired
	}
	lang := strings.ToLower(language)
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}
	return s.getCached(ctx, siteID, researchTopicHash(topic), lang)
}

// DeepResearchSummary runs deep research and returns the pipeline-ready
// summary (used as PipelineInput.ResearchFn). Always returns a non-nil
// summary; errors are wrapped.
func (s *Service) DeepResearchSummary(ctx context.Context, siteID uuid.UUID, topic, language string) (*ai.ResearchSummary, error) {
	report, err := s.DeepResearch(ctx, siteID, topic, language)
	if err != nil {
		return nil, err
	}
	return s.summaryFromReport(report), nil
}

// GetCachedSummary is the cache-only lookup used by the Translation Engine:
// it reuses an existing research without triggering a new search.
func (s *Service) GetCachedSummary(ctx context.Context, siteID uuid.UUID, topic, language string) (*ai.ResearchSummary, error) {
	cached, err := s.GetCachedResearch(ctx, siteID, topic, language)
	if err != nil {
		return nil, err
	}
	return &ai.ResearchSummary{
		Topic:    cached.Topic,
		Language: cached.Language,
		Briefing: formatCachedBriefing(cached),
		Facts:    aiFactsFromEntries(cached.Facts),
		Sources:  aiSourcesFromEntries(cached.Sources),
		Cached:   true,
	}, nil
}

func (s *Service) summaryFromReport(report *DeepResearchReport) *ai.ResearchSummary {
	summary := &ai.ResearchSummary{
		Topic:    report.Topic,
		Language: report.Language,
		Cached:   report.Cached,
	}
	if report.Briefing != nil {
		summary.Briefing = formatBriefingDoc(*report.Briefing)
	}
	summary.Facts = aiFactsFromEntries(report.Facts)
	summary.Sources = aiSourcesFromEntries(report.Sources)
	return summary
}

func aiFactsFromEntries(entries []FactBaseEntry) []ai.ResearchFact {
	var out []ai.ResearchFact
	for _, f := range entries {
		out = append(out, ai.ResearchFact{
			Type:       string(f.FactType),
			Entity:     f.Entity,
			Value:      f.Value,
			Source:     f.SourceURL,
			Confidence: f.Confidence,
		})
	}
	return out
}

func aiSourcesFromEntries(sources []ResearchSource) []ai.ResearchSourceSummary {
	var out []ai.ResearchSourceSummary
	for _, src := range sources {
		out = append(out, ai.ResearchSourceSummary{
			Title:            src.Title,
			URL:              src.URL,
			Domain:           src.Domain,
			ReliabilityScore: src.ReliabilityScore,
			ReliabilityLabel: src.ReliabilityLabel,
		})
	}
	return out
}

// decorateSources adds domain, reliability score and label to grounding sources.
func (s *Service) decorateSources(sources []ResearchSource) []ResearchSource {
	for i := range sources {
		domain := ai.ExtractDomain(sources[i].URL)
		score, _ := ai.ReliabilityOfDomain(domain)
		sources[i].Domain = domain
		sources[i].ReliabilityScore = score
		sources[i].ReliabilityLabel = ai.ReliabilityLabel(score)
	}
	return sources
}

// --- persistence helpers ---

func (s *Service) getCached(ctx context.Context, siteID uuid.UUID, hash, language string) (*CachedResearch, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var c CachedResearch
	var briefingStr, factsStr, sourcesStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, topic, language, COALESCE(briefing::text, '{}'),
		        COALESCE(fact_base::text, '[]'), COALESCE(sources::text, '[]'),
		        hit_count, created_at, expires_at
		 FROM research_cache
		 WHERE site_id = $1 AND topic_hash = $2 AND language = $3 AND expires_at > NOW()`,
		siteID, hash, language,
	).Scan(&c.ID, &c.SiteID, &c.Topic, &c.Language, &briefingStr, &factsStr, &sourcesStr,
		&c.HitCount, &c.CreatedAt, &c.ExpiresAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCacheEntryNotFound
		}
		return nil, err
	}

	_ = json.Unmarshal([]byte(briefingStr), &c.Briefing)
	_ = json.Unmarshal([]byte(factsStr), &c.Facts)
	_ = json.Unmarshal([]byte(sourcesStr), &c.Sources)

	_, _ = p.Exec(ctx,
		`UPDATE research_cache SET hit_count = hit_count + 1 WHERE id = $1`, c.ID)

	return &c, nil
}

func (s *Service) saveFactBase(ctx context.Context, facts []FactBaseEntry) error {
	if len(facts) == 0 {
		return nil
	}
	p, err := s.pool()
	if err != nil {
		return err
	}
	for i := range facts {
		if facts[i].ID == uuid.Nil {
			facts[i].ID = uuid.New()
		}
		if facts[i].CreatedAt.IsZero() {
			facts[i].CreatedAt = time.Now()
		}
		_, err = p.Exec(ctx,
			`INSERT INTO research_fact_base (id, site_id, research_job_id, fact_type, entity, value, source_url, confidence, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			facts[i].ID, facts[i].SiteID, facts[i].ResearchJobID,
			string(facts[i].FactType), facts[i].Entity, facts[i].Value,
			facts[i].SourceURL, facts[i].Confidence, facts[i].CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert fact base entry: %w", err)
		}
	}
	return nil
}

func (s *Service) persistBriefing(ctx context.Context, jobID uuid.UUID, doc ResearchBriefingDoc) error {
	asMap := map[string]interface{}{}
	raw, _ := json.Marshal(doc)
	_ = json.Unmarshal(raw, &asMap)

	_, err := s.SaveBriefing(ctx, jobID, ResearchBriefing{
		ResearchJobID:      jobID,
		StructuredBriefing: asMap,
		Timeline:           []interface{}{},
		ConfirmedFacts:     []interface{}{},
		ConflictingInfo:    []interface{}{},
		EditorialApproaches: []interface{}{},
	})
	return err
}

func (s *Service) saveCache(ctx context.Context, siteID uuid.UUID, topic, hash, language string,
	briefing ResearchBriefingDoc, facts []FactBaseEntry, sources []ResearchSource) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	ttl := s.cacheTTL
	if ttl <= 0 {
		ttl = DefaultResearchCacheTTL
	}
	now := time.Now()

	briefingJSON, _ := json.Marshal(briefing)
	factsJSON, _ := json.Marshal(facts)
	sourcesJSON, _ := json.Marshal(sources)

	_, err = p.Exec(ctx,
		`INSERT INTO research_cache (id, site_id, topic, topic_hash, language, briefing, fact_base, sources, hit_count, created_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,0,$9,$10)
		 ON CONFLICT (site_id, topic_hash, language) DO UPDATE SET
		   briefing = EXCLUDED.briefing,
		   fact_base = EXCLUDED.fact_base,
		   sources = EXCLUDED.sources,
		   expires_at = EXCLUDED.expires_at,
		   created_at = EXCLUDED.created_at`,
		uuid.New(), siteID, topic, hash, language,
		string(briefingJSON), string(factsJSON), string(sourcesJSON),
		now, now.Add(ttl),
	)
	return err
}

func (s *Service) reportFromCache(c *CachedResearch) *DeepResearchReport {
	return &DeepResearchReport{
		ResearchJob: ResearchJob{
			Topic:        c.Topic,
			Language:     c.Language,
			Status:       JobStatusCompleted,
			SourcesCount: len(c.Sources),
		},
		Briefing: &c.Briefing,
		Facts:    c.Facts,
		Sources:  c.Sources,
		Cached:   true,
	}
}

func formatCachedBriefing(c *CachedResearch) string {
	return formatBriefingDoc(c.Briefing)
}

func formatBriefingDoc(doc ResearchBriefingDoc) string {
	var b strings.Builder
	b.WriteString("Topic: ")
	b.WriteString(doc.Topic)
	if doc.Summary != "" {
		b.WriteString("\nSummary: ")
		b.WriteString(doc.Summary)
	}
	writeSection := func(name string, items []string) {
		if len(items) == 0 {
			return
		}
		b.WriteString("\n")
		b.WriteString(name)
		b.WriteString(":")
		for _, it := range items {
			b.WriteString("\n- ")
			b.WriteString(it)
		}
	}
	writeSection("Key points", doc.KeyPoints)
	writeSection("Data found", doc.DataFound)
	writeSection("Statistics", doc.Statistics)
	writeSection("Dates", doc.Dates)
	writeSection("Companies", doc.Companies)
	writeSection("Products", doc.Products)
	writeSection("Conclusions", doc.Conclusions)
	return b.String()
}
