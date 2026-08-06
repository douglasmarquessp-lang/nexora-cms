-- AI Editorial Brain (Sprint 5.11)
-- Editorial briefs (intent/persona/outline/questions), editorial reviews
-- (final note + decision), per-block confidence, and claim→source evidence.

CREATE TABLE IF NOT EXISTS editorial_briefs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    topic VARCHAR(500) NOT NULL,
    topic_hash VARCHAR(64) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    search_intent VARCHAR(20) NOT NULL,
    intent_confidence REAL NOT NULL DEFAULT 0,
    persona VARCHAR(20) NOT NULL,
    persona_confidence REAL NOT NULL DEFAULT 0,
    audience VARCHAR(255) NOT NULL,
    angle VARCHAR(500) NOT NULL,
    suggested_title VARCHAR(500) NOT NULL,
    outline JSONB DEFAULT '[]'::jsonb,
    questions JSONB DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ready',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_editorial_briefs_site_topic ON editorial_briefs(site_id, topic_hash, language);
CREATE INDEX IF NOT EXISTS idx_editorial_briefs_status ON editorial_briefs(status);

CREATE TABLE IF NOT EXISTS editorial_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    brief_id UUID,
    article_id UUID,
    article_title VARCHAR(500) NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    seo_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    eeat_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    freshness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    coverage_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    naturalness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    confidence_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    final_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    decision VARCHAR(20) NOT NULL DEFAULT 'needs_review',
    threshold NUMERIC(5,2) NOT NULL DEFAULT 90,
    coverage JSONB DEFAULT '{}'::jsonb,
    fluency JSONB DEFAULT '{}'::jsonb,
    semantic JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_editorial_reviews_site_hash ON editorial_reviews(site_id, content_hash);
CREATE INDEX IF NOT EXISTS idx_editorial_reviews_article ON editorial_reviews(article_id);
CREATE INDEX IF NOT EXISTS idx_editorial_reviews_decision ON editorial_reviews(decision);

CREATE TABLE IF NOT EXISTS editorial_block_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES editorial_reviews(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    block VARCHAR(255) NOT NULL,
    score NUMERIC(5,2) NOT NULL DEFAULT 0,
    evidence_count INT NOT NULL DEFAULT 0,
    note VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_editorial_block_scores_review ON editorial_block_scores(review_id);

CREATE TABLE IF NOT EXISTS editorial_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES editorial_reviews(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    claim TEXT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    source_title VARCHAR(500) NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    confidence NUMERIC(5,2) NOT NULL DEFAULT 0,
    note VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_editorial_evidence_review ON editorial_evidence(review_id);

ALTER TABLE editorial_briefs ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_block_scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_evidence ENABLE ROW LEVEL SECURITY;

CREATE POLICY editorial_briefs_isolation ON editorial_briefs
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY editorial_reviews_isolation ON editorial_reviews
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY editorial_block_scores_isolation ON editorial_block_scores
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY editorial_evidence_isolation ON editorial_evidence
    USING (site_id = current_setting('app.current_site_id')::UUID);

DROP TRIGGER IF EXISTS trg_editorial_briefs_updated ON editorial_briefs;
CREATE TRIGGER trg_editorial_briefs_updated
    BEFORE UPDATE ON editorial_briefs
    FOR EACH ROW
    WHEN (pg_trigger_depth() = 0)
    EXECUTE FUNCTION update_updated_at_column();
