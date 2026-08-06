-- Sprint 5.8 — AI Research Intelligence
-- research_cache: 24h cache of deep research results keyed by (site, topic hash, language)
CREATE TABLE IF NOT EXISTS research_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    topic VARCHAR(500) NOT NULL,
    topic_hash VARCHAR(64) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    briefing JSONB DEFAULT '{}'::jsonb,
    fact_base JSONB DEFAULT '[]'::jsonb,
    sources JSONB DEFAULT '[]'::jsonb,
    hit_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_research_cache_site_topic_lang ON research_cache(site_id, topic_hash, language);
CREATE INDEX IF NOT EXISTS idx_research_cache_expires ON research_cache(expires_at);

-- research_fact_base: structured facts (company, product, version, price, date, event, technology, number)
CREATE TABLE IF NOT EXISTS research_fact_base (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    research_job_id UUID NOT NULL REFERENCES research_jobs(id) ON DELETE CASCADE,
    fact_type VARCHAR(50) NOT NULL,
    entity VARCHAR(500) NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    source_url TEXT DEFAULT '',
    confidence NUMERIC(5,2) NOT NULL DEFAULT 50,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_research_fact_base_job ON research_fact_base(research_job_id);
CREATE INDEX IF NOT EXISTS idx_research_fact_base_type ON research_fact_base(fact_type);

-- Reliability ranking columns for research sources
ALTER TABLE research_sources
ADD COLUMN IF NOT EXISTS domain VARCHAR(255) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS reliability_score INT NOT NULL DEFAULT 0;

-- RLS policies (site isolation), following the 000016/000025 pattern
ALTER TABLE research_cache ENABLE ROW LEVEL SECURITY;
ALTER TABLE research_fact_base ENABLE ROW LEVEL SECURITY;

CREATE POLICY research_cache_isolation ON research_cache
    USING (site_id = current_setting('app.current_site_id')::UUID);

CREATE POLICY research_fact_base_isolation ON research_fact_base
    USING (site_id = current_setting('app.current_site_id')::UUID);
