CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

ALTER TABLE seo_projects
    ADD COLUMN IF NOT EXISTS eeat_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS freshness_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS topical_authority_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS slug_target VARCHAR(500) DEFAULT '',
    ADD COLUMN IF NOT EXISTS meta_title_target VARCHAR(200) DEFAULT '',
    ADD COLUMN IF NOT EXISTS meta_description_target TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS checklist JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS ai_suggestions JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS content_type VARCHAR(50) DEFAULT 'article';

ALTER TABLE seo_keywords
    ADD COLUMN IF NOT EXISTS cluster_id UUID REFERENCES seo_clusters(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cannibalization_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS content_gap_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS semantic_entities TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS topical_relevance NUMERIC(5,2) DEFAULT 0;

ALTER TABLE seo_clusters
    ADD COLUMN IF NOT EXISTS topical_authority_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS semantic_entities TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS internal_links_count INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS content_gap_articles TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS parent_cluster_id UUID REFERENCES seo_clusters(id) ON DELETE SET NULL;

ALTER TABLE seo_audits
    ADD COLUMN IF NOT EXISTS eeat_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS freshness_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS heading_issues JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS image_alt_issues JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS schema_issues JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS slug_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS slug_issues TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS title_issues TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS meta_issues TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS orphan_detected BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS cannibalization_detected BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS content_gap_detected BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS link_suggestions JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS checklist_items JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS eeat_issues JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS freshness_issues JSONB DEFAULT '[]'::jsonb;

ALTER TABLE seo_scores
    ADD COLUMN IF NOT EXISTS eeat_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS freshness_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS topical_authority_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS schema_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS image_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS slug_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS heading_score NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS multilingual_score NUMERIC(5,2) DEFAULT 0;

CREATE TABLE IF NOT EXISTS seo_improvements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_project_id UUID REFERENCES seo_projects(id) ON DELETE CASCADE,
    post_id UUID,
    category VARCHAR(100) NOT NULL,
    issue VARCHAR(500) NOT NULL,
    suggestion TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    impact_score NUMERIC(5,2) DEFAULT 0,
    effort_score NUMERIC(5,2) DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    applied_at TIMESTAMPTZ,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_seo_improvements_site ON seo_improvements(site_id);
CREATE INDEX IF NOT EXISTS idx_seo_improvements_project ON seo_improvements(seo_project_id);
CREATE INDEX IF NOT EXISTS idx_seo_improvements_category ON seo_improvements(category);
CREATE INDEX IF NOT EXISTS idx_seo_improvements_status ON seo_improvements(status);
CREATE INDEX IF NOT EXISTS idx_seo_improvements_priority ON seo_improvements(priority);
CREATE INDEX IF NOT EXISTS idx_seo_kw_cluster ON seo_keywords(cluster_id);
