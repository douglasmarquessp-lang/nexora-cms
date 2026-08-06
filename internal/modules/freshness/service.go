package freshness

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/kernel"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

// SourceInput is the minimal data needed to freshness-score a source.
type SourceInput struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	Text        string     `json:"text,omitempty"`
}

// ArticleForSweep is one published article passed to the daily refresh sweep.
type ArticleForSweep struct {
	PublicationID uuid.UUID  `json:"publication_id"`
	Topic         string     `json:"topic"`
	Content       string     `json:"content"`
	Language      string     `json:"language"`
	PublishedAt   time.Time  `json:"published_at"`
	Intent        IntentType `json:"intent"`
}

// Service is the freshness engine: intent classification, temporal windows,
// per-source scoring, version history, dedupe and the daily update sweep.
type Service struct {
	log           *logger.Logger
	db            *database.Database
	eventBus      *kernel.EventBus
	sweepEnabled  bool
	newsMaxDays   int
	newsNeverOlderDays int
}

// NewService builds the freshness engine service.
func NewService(cfg *config.Config, log *logger.Logger, db *database.Database) *Service {
	s := &Service{log: log, db: db, sweepEnabled: true, newsMaxDays: 30, newsNeverOlderDays: 90}
	if cfg != nil {
		s.sweepEnabled = cfg.Freshness.SweepEnabled
		if cfg.Freshness.NewsMaxDays > 0 {
			s.newsMaxDays = cfg.Freshness.NewsMaxDays
		}
		if cfg.Freshness.NewsNeverOlderDays > 0 {
			s.newsNeverOlderDays = cfg.Freshness.NewsNeverOlderDays
		}
	}
	return s
}

// windowFor returns the temporal window for an intent honoring configured
// overrides (defaults match WindowStrategy).
func (s *Service) windowFor(intent IntentType) TemporalWindow {
	w := WindowStrategy(intent)
	if intent == IntentNews {
		if s.newsMaxDays > 0 {
			w.MaxDays = s.newsMaxDays
		}
		if s.newsNeverOlderDays > 0 {
			w.NeverOlderDays = s.newsNeverOlderDays
		}
	}
	return w
}

func (s *Service) SetEventBus(bus *kernel.EventBus) { s.eventBus = bus }

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

func (s *Service) fireEvent(event kernel.EventType, payload interface{}, siteID string) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.EmitAsync(context.Background(), event, payload, siteID)
}

// ClassifyAndWindow classifies the intent deterministically and — when a pool
// is available — caches the classification into news_intents. Returns the
// classification and its derived temporal window.
func (s *Service) ClassifyAndWindow(ctx context.Context, siteID uuid.UUID, topic, content, lang string) (IntentResult, TemporalWindow, error) {
	ir, err := ClassifyIntent(topic, content, lang)
	if err != nil {
		return IntentResult{}, TemporalWindow{}, err
	}
	win := WindowStrategy(ir.Intent)

	p, err := s.pool()
	if err != nil {
		return ir, win, nil // DB-off: still return the deterministic result
	}

	signals, _ := json.Marshal(ir.Signals)
	hash := Fingerprint(topic, content)
	_, err = p.Exec(ctx, `
INSERT INTO news_intents (site_id, topic, topic_hash, language, intent, confidence, signals,
	window_recent_days, window_max_days, never_older_days, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,NOW())
ON CONFLICT (site_id, topic_hash, language) DO UPDATE SET
	intent=EXCLUDED.intent, confidence=EXCLUDED.confidence,
	signals=EXCLUDED.signals,
	window_recent_days=EXCLUDED.window_recent_days,
	window_max_days=EXCLUDED.window_max_days,
	never_older_days=EXCLUDED.never_older_days`,
		siteID, topic, hash, lang, string(ir.Intent), ir.Confidence, string(signals),
		win.RecentDays, win.MaxDays, win.NeverOlderDays)
	if err != nil {
		s.log.Warn("failed to cache intent", "error", err)
	}
	s.fireEvent(EventIntentClassified, ir, siteID.String())
	return ir, win, nil
}

// ScoreSources computes the per-source freshness breakdown for an explicit
// intent (already classified), marks obsolete entities, sorts by priority and
// score, and persists source_freshness_scores when a pool is available.
func (s *Service) ScoreSources(ctx context.Context, siteID uuid.UUID, researchJobID *uuid.UUID, intent IntentType, targetEntity string, entities []EntityVersion, sources []SourceInput) ([]FreshnessBreakdown, error) {
	if len(sources) == 0 {
		return nil, ErrSourceRequired
	}
	now := time.Now().UTC()

	scored := make([]FreshnessBreakdown, 0, len(sources))
	for _, src := range sources {
		br := ComputeSourceFreshnessWithWindow(now, src.PublishedAt, src.UpdatedAt, s.windowFor(intent), src.Title, src.URL, targetEntity)
		if len(entities) > 0 {
			for _, oc := range DetectObsoleteSources(src.Text+" "+src.Title, entities) {
				br.Obsolete = true
				br.ObsoleteEntity = oc.Entity
				br.Reasons = append(br.Reasons, "obsolescence: "+oc.Entity)
			}
		}
		scored = append(scored, br)
	}
	scored = SortSourcesByPriorityAndScore(scored)

	if p, err := s.pool(); err == nil {
		for _, br := range scored {
			var jobID interface{}
			if researchJobID != nil {
				jobID = *researchJobID
			}
			reasons, _ := json.Marshal(br.Reasons)
			_, _ = p.Exec(ctx, `
INSERT INTO source_freshness_scores (site_id, research_job_id, source_url, intent,
	published_at, updated_at, age_days, freshness_score,
	age_component, update_component, source_component,
	source_priority, obsolete, usable, reasons, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,NOW())`,
				siteID, jobID, br.SourceURL, string(intent),
				br.PublishedAt, br.UpdatedAt, br.AgeDays, br.Score,
				br.AgeComponent, br.UpdateComponent, br.SourceComponent,
				string(br.SourcePriority), br.Obsolete, br.Usable, string(reasons))
		}
	}

	s.fireEvent(EventSourcesScored, scored, siteID.String())
	return scored, nil
}

// DetectObsolete returns the obsolete-info flags for a supplied text.
func (s *Service) DetectObsolete(text string, entities []EntityVersion) []ObsoleteCheck {
	if len(entities) == 0 {
		return nil
	}
	return DetectObsoleteSources(text, entities)
}

// SaveVersion persists a version record. Returns ErrDatabaseNotAvail when no
// pool is configured.
func (s *Service) SaveVersion(ctx context.Context, siteID uuid.UUID, v VersionRecord) error {
	p, err := s.pool()
	if err != nil {
		return err
	}
	changes, _ := json.Marshal(v.Changes)
	diff, _ := json.Marshal(v.Diff)
	sources, _ := json.Marshal(v.Sources)
	_, err = p.Exec(ctx, `
INSERT INTO article_versions (site_id, publication_id, version, intent, changes, diff, sources, created_at)
VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7::jsonb,NOW())`,
		siteID, v.PublicationID, v.Version, string(v.Intent),
		string(changes), string(diff), string(sources))
	if err != nil {
		return err
	}
	s.fireEvent(EventVersionSaved, v, siteID.String())
	return nil
}

// ListVersions returns the stored versions for one publication, latest first.
func (s *Service) ListVersions(ctx context.Context, siteID, pubID uuid.UUID) ([]VersionRecord, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx,
		`SELECT version, intent, changes, diff, sources, created_at
		 FROM article_versions WHERE site_id=$1 AND publication_id=$2
		 ORDER BY created_at DESC`, siteID, pubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VersionRecord
	for rows.Next() {
		var v VersionRecord
		var intent string
		var changes, diff, sources []byte
		var created time.Time
		if err := rows.Scan(&v.Version, &intent, &changes, &diff, &sources, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(changes, &v.Changes)
		_ = json.Unmarshal(diff, &v.Diff)
		_ = json.Unmarshal(sources, &v.Sources)
		v.Intent = IntentType(intent)
		v.CreatedAt = created
		v.PublicationID = pubID
		out = append(out, v)
	}
	return out, rows.Err()
}

// CheckDuplicate verifies whether a topic was already covered the same day
// (news_dedup) and registers the fingerprint so future lookups can match.
// When a duplicate is found the caller must UPDATE the existing article
// instead of creating a new one.
func (s *Service) CheckDuplicate(ctx context.Context, siteID uuid.UUID, topic, content, lang string, pubID uuid.UUID, now time.Time) (DedupCandidate, error) {
	fp := Fingerprint(topic, content)
	dc := DedupCandidate{
		Fingerprint: fp,
		Topic:       topic,
		Language:    lang,
		Intent:      classifyForDedup(topic, lang),
	}

	p, err := s.pool()
	if err != nil {
		return dc, nil // DB-off: result stays "no match"
	}

	rows, err := p.Query(ctx,
		`SELECT publication_id, topic, created_on FROM news_dedup WHERE site_id=$1`, siteID)
	if err != nil {
		return dc, err
	}
	defer rows.Close()
	var candidates []Candidate
	for rows.Next() {
		var c Candidate
		var created time.Time
		if err := rows.Scan(&c.PublicationID, &c.Topic, &created); err != nil {
			continue
		}
		c.PublishedAt = &created
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return dc, err
	}

	check := CheckDuplicate(topic, content, lang, now, candidates)
	check.Fingerprint = fp
	check.Intent = dc.Intent
	dc.Duplicate = check.Duplicate
	dc.SameDay = check.SameDay
	dc.MatchRatio = check.MatchRatio
	dc.ExistingPubID = check.ExistingPubID
	dc.ExistingDate = check.ExistingDate

	if !dc.Duplicate {
		var jid interface{}
		if pubID != uuid.Nil {
			jid = pubID
		}
		_, err = p.Exec(ctx, `
INSERT INTO news_dedup (site_id, fingerprint, topic, language, intent, publication_id, created_on, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())`,
			siteID, fp, topic, lang, string(dc.Intent), jid, now.Format("2006-01-02"))
		if err != nil {
			return dc, err
		}
	}
	if dc.Duplicate {
		s.fireEvent(EventDedupFound, dc, siteID.String())
	}
	return dc, nil
}

// RunDailySweep implements the once-per-day re-evaluation of published
// articles. The sweep is skipped when it already ran today (freshness_sweeps
// guard). Articles that fall outside their temporal window or that have a
// newer researched source are flagged Needs Update in content_updates.
func (s *Service) RunDailySweep(ctx context.Context, siteID uuid.UUID, articles []ArticleForSweep) ([]SweepDecision, error) {
	now := time.Now().UTC()
	if !s.sweepEnabled {
		return nil, ErrSweepDisabled
	}
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	// Once-per-day guard: skip when the sweep already ran today.
	var lastRun time.Time
	hasRun := true
	row := p.QueryRow(ctx, `SELECT last_run_at FROM freshness_sweeps WHERE site_id=$1`, siteID)
	if err := row.Scan(&lastRun); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			hasRun = false
		} else {
			s.log.Warn("failed to read sweep guard", "error", err)
		}
	}
	var guardPtr *time.Time
	if hasRun {
		guardPtr = &lastRun
	}
	if !DailySweepOnce(guardPtr, now) {
		return nil, ErrSweepAlreadyRun
	}

	decisions := make([]SweepDecision, 0, len(articles))
	for _, a := range articles {
		hasNewer := false
		// Newer researched source for the same topic exists?
		rows, err := p.Query(ctx,
			`SELECT 1 FROM news_dedup WHERE site_id=$1 AND created_on > $2 AND topic = $3 LIMIT 1`,
			siteID, a.PublishedAt.Format("2006-01-02"), a.Topic)
		if err == nil {
			hasNewer = rows.Next()
			rows.Close()
		}
		dec := ReEvaluateArticleWithWindow(a.PublishedAt, a.Intent, now, hasNewer, s.windowFor(a.Intent))
		dec.PublicationID = a.PublicationID
		if dec.NeedsUpdate {
			details, _ := json.Marshal(dec.Details)
			_, _ = p.Exec(ctx, `
INSERT INTO content_updates (site_id, publication_id, intent, reason, old_score, new_score, details, status, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,NOW())
ON CONFLICT DO NOTHING`,
				siteID, a.PublicationID, string(a.Intent), dec.Reason, dec.OldScore, dec.NewScore, string(details), string(UpdateNeedsWork))
			s.fireEvent(EventContentNeedsUpdate, dec, siteID.String())
		}
		decisions = append(decisions, dec)
	}

	_, err = p.Exec(ctx, `
INSERT INTO freshness_sweeps (site_id, last_run_at) VALUES ($1,NOW())
ON CONFLICT (site_id) DO UPDATE SET last_run_at=NOW()`, siteID)
	if err != nil {
		s.log.Warn("failed to record sweep", "error", err)
	}

	s.fireEvent(EventSweepCompleted, decisions, siteID.String())
	return decisions, nil
}

// ListUpdates returns the Needs Update flags for a site.
func (s *Service) ListUpdates(ctx context.Context, siteID uuid.UUID) ([]ContentUpdate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx,
		`SELECT id, publication_id, intent, reason, old_score, new_score, details, status, created_at, resolved_at
		 FROM content_updates WHERE site_id=$1 ORDER BY created_at DESC`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentUpdate
	for rows.Next() {
		var u ContentUpdate
		var intent, status string
		var details []byte
		var resolved *time.Time
		if err := rows.Scan(&u.ID, &u.PublicationID, &intent, &u.Reason, &u.OldScore, &u.NewScore, &details, &status, &u.CreatedAt, &resolved); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(details, &u.Details)
		u.Intent = IntentType(intent)
		u.Status = UpdateStatus(status)
		u.SiteID = siteID
		u.ResolvedAt = resolved
		out = append(out, u)
	}
	return out, rows.Err()
}
