-- Freshness Engine + News Intelligence (Sprint 5.10)
-- Intent caching, per-source freshness scoring, version history,
-- news dedupe, and content-update tracking.

CREATE TABLE IF NOT EXISTS news_intents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    topic VARCHAR(500) NOT NULL,
    topic_hash VARCHAR(64) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    intent VARCHAR(20) NOT NULL,
    confidence REAL NOT NULL DEFAULT 0,
    signals JSONB DEFAULT '[]'::jsonb,
    window_recent_days INT NOT NULL DEFAULT 0,
    window_max_days INT NOT NULL DEFAULT 0,
    never_older_days INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_news_intents_site_topic_lang ON news_intents(site_id, topic_hash, language);
CREATE INDEX IF NOT EXISTS idx_news_intents_intent ON news_intents(intent);

CREATE TABLE IF NOT EXISTS source_freshness_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    research_job_id UUID,
    source_url TEXT NOT NULL,
    intent VARCHAR(20) NOT NULL DEFAULT 'evergreen',
    published_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    age_days INT NOT NULL DEFAULT 0,
    freshness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    age_component NUMERIC(5,2) NOT NULL DEFAULT 0,
    update_component NUMERIC(5,2) NOT NULL DEFAULT 0,
    source_component NUMERIC(5,2) NOT NULL DEFAULT 0,
    source_priority VARCHAR(20) NOT NULL DEFAULT 'other',
    obsolete BOOLEAN NOT NULL DEFAULT FALSE,
    usable BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_source_freshness_job ON source_freshness_scores(research_job_id);

CREATE TABLE IF NOT EXISTS article_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    publication_id UUID,
    version VARCHAR(20) NOT NULL,
    intent VARCHAR(20) NOT NULL DEFAULT 'evergreen',
    changes JSONB DEFAULT '[]'::jsonb,
    diff JSONB DEFAULT '[]'::jsonb,
    sources JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_article_versions_pub ON article_versions(publication_id, version);

CREATE TABLE IF NOT EXISTS news_dedup (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    fingerprint VARCHAR(64) NOT NULL,
    topic VARCHAR(500) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    intent VARCHAR(20) NOT NULL DEFAULT 'evergreen',
    publication_id UUID,
    created_on DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_news_dedup_site_fingerprint ON news_dedup(site_id, fingerprint);

CREATE TABLE IF NOT EXISTS content_updates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    publication_id UUID,
    intent VARCHAR(20) NOT NULL DEFAULT 'evergreen',
    reason VARCHAR(30) NOT NULL DEFAULT 'freshness',
    old_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    new_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    details JSONB DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_updates_status ON content_updates(status);

CREATE TABLE IF NOT EXISTS freshness_sweeps (
    site_id UUID PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
    last_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE news_intents ENABLE ROW LEVEL SECURITY;
ALTER TABLE source_freshness_scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE news_dedup ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_updates ENABLE ROW LEVEL SECURITY;
ALTER TABLE freshness_sweeps ENABLE ROW LEVEL SECURITY;

CREATE POLICY news_intents_isolation ON news_intents
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY source_freshness_scores_isolation ON source_freshness_scores
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY article_versions_isolation ON article_versions
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY news_dedup_isolation ON news_dedup
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY content_updates_isolation ON content_updates
    USING (site_id = current_setting('app.current_site_id')::UUID);
CREATE POLICY freshness_sweeps_isolation ON freshness_sweeps
    USING (site_id = current_setting('app.current_site_id')::UUID);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column') THEN
        CREATE TRIGGER trg_freshness_news_intents_updated_at
            BEFORE UPDATE ON news_intents
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
        CREATE TRIGGER trg_freshness_content_updates_updated_at
            BEFORE UPDATE ON content_updates
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;