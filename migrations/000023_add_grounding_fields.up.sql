-- Add grounding-related columns to research_sources
ALTER TABLE research_sources
ADD COLUMN IF NOT EXISTS freshness_score NUMERIC(5,4) DEFAULT 0,
ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS retrieved_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS grounding_metadata JSONB DEFAULT '{}'::jsonb;

-- Create article_sources table: links generated content to its supporting sources
CREATE TABLE IF NOT EXISTS article_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    article_id UUID,
    pipeline_job_id UUID,
    workflow_job_id UUID,
    autocontent_job_id UUID,
    source_url TEXT NOT NULL,
    title VARCHAR(500) DEFAULT '',
    snippet TEXT DEFAULT '',
    language VARCHAR(10) DEFAULT '',
    author VARCHAR(255) DEFAULT '',
    published_at TIMESTAMPTZ,
    retrieved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    freshness_score NUMERIC(5,4) DEFAULT 0,
    is_verified BOOLEAN DEFAULT false,
    domain_rank INT DEFAULT 0,
    relevance_score INT DEFAULT 0,
    grounding_metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_article_sources_site ON article_sources(site_id);
CREATE INDEX IF NOT EXISTS idx_article_sources_article ON article_sources(article_id);
CREATE INDEX IF NOT EXISTS idx_article_sources_pipeline ON article_sources(pipeline_job_id);
CREATE INDEX IF NOT EXISTS idx_article_sources_workflow ON article_sources(workflow_job_id);
CREATE INDEX IF NOT EXISTS idx_article_sources_url ON article_sources(source_url);
CREATE INDEX IF NOT EXISTS idx_article_sources_verified ON article_sources(is_verified);
CREATE INDEX IF NOT EXISTS idx_article_sources_freshness ON article_sources(freshness_score);
