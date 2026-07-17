CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS seo_projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id UUID,
    title VARCHAR(500) NOT NULL,
    target_url VARCHAR(2048) DEFAULT '',
    post_id UUID,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    seo_score NUMERIC(5,2) DEFAULT 0,
    readability_score NUMERIC(5,2) DEFAULT 0,
    keyword_density NUMERIC(5,2) DEFAULT 0,
    content_quality NUMERIC(5,2) DEFAULT 0,
    technical_score NUMERIC(5,2) DEFAULT 0,
    recommendations TEXT[] DEFAULT '{}',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT DEFAULT '',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_projects_site ON seo_projects(site_id);
CREATE INDEX idx_seo_projects_status ON seo_projects(status);
CREATE INDEX idx_seo_projects_post ON seo_projects(post_id);

CREATE TABLE IF NOT EXISTS seo_keywords (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_project_id UUID REFERENCES seo_projects(id) ON DELETE CASCADE,
    keyword VARCHAR(500) NOT NULL,
    keyword_type VARCHAR(50) NOT NULL DEFAULT 'primary',
    search_intent VARCHAR(50) DEFAULT 'informational',
    volume INT DEFAULT 0,
    difficulty NUMERIC(5,2) DEFAULT 0,
    density NUMERIC(5,2) DEFAULT 0,
    frequency INT DEFAULT 0,
    prominence NUMERIC(5,2) DEFAULT 0,
    entities TEXT[] DEFAULT '{}',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_kw_site ON seo_keywords(site_id);
CREATE INDEX idx_seo_kw_project ON seo_keywords(seo_project_id);
CREATE INDEX idx_seo_kw_type ON seo_keywords(keyword_type);
CREATE INDEX idx_seo_kw_intent ON seo_keywords(search_intent);

CREATE TABLE IF NOT EXISTS seo_clusters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name VARCHAR(300) NOT NULL,
    description TEXT DEFAULT '',
    keywords TEXT[] DEFAULT '{}',
    article_count INT DEFAULT 0,
    avg_score NUMERIC(5,2) DEFAULT 0,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_clusters_site ON seo_clusters(site_id);

CREATE TABLE IF NOT EXISTS seo_audits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_project_id UUID REFERENCES seo_projects(id) ON DELETE CASCADE,
    post_id UUID,
    url VARCHAR(2048) DEFAULT '',
    title_score NUMERIC(5,2) DEFAULT 0,
    meta_score NUMERIC(5,2) DEFAULT 0,
    heading_score NUMERIC(5,2) DEFAULT 0,
    paragraph_score NUMERIC(5,2) DEFAULT 0,
    readability_score NUMERIC(5,2) DEFAULT 0,
    passive_voice_score NUMERIC(5,2) DEFAULT 0,
    sentence_variation_score NUMERIC(5,2) DEFAULT 0,
    duplicate_score NUMERIC(5,2) DEFAULT 0,
    overall_score NUMERIC(5,2) DEFAULT 0,
    issues JSONB DEFAULT '[]'::jsonb,
    recommendations TEXT[] DEFAULT '{}',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    audited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_audits_site ON seo_audits(site_id);
CREATE INDEX idx_seo_audits_project ON seo_audits(seo_project_id);
CREATE INDEX idx_seo_audits_post ON seo_audits(post_id);

CREATE TABLE IF NOT EXISTS seo_internal_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_project_id UUID REFERENCES seo_projects(id) ON DELETE CASCADE,
    source_url VARCHAR(2048) NOT NULL,
    target_url VARCHAR(2048) NOT NULL,
    anchor_text VARCHAR(500) DEFAULT '',
    link_type VARCHAR(50) NOT NULL DEFAULT 'suggestion',
    relevance NUMERIC(5,2) DEFAULT 0,
    is_existing BOOLEAN DEFAULT false,
    is_implemented BOOLEAN DEFAULT false,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_links_site ON seo_internal_links(site_id);
CREATE INDEX idx_seo_links_project ON seo_internal_links(seo_project_id);
CREATE INDEX idx_seo_links_source ON seo_internal_links(source_url);

CREATE TABLE IF NOT EXISTS seo_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_project_id UUID REFERENCES seo_projects(id) ON DELETE CASCADE,
    post_id UUID,
    title_tag VARCHAR(200) DEFAULT '',
    meta_description TEXT DEFAULT '',
    canonical_url VARCHAR(2048) DEFAULT '',
    og_title VARCHAR(200) DEFAULT '',
    og_description TEXT DEFAULT '',
    og_image VARCHAR(2048) DEFAULT '',
    twitter_title VARCHAR(200) DEFAULT '',
    twitter_description TEXT DEFAULT '',
    twitter_image VARCHAR(2048) DEFAULT '',
    json_ld JSONB DEFAULT '{}'::jsonb,
    faq_schema JSONB DEFAULT '[]'::jsonb,
    breadcrumb_schema JSONB DEFAULT '[]'::jsonb,
    article_schema JSONB DEFAULT '{}'::jsonb,
    hreflang JSONB DEFAULT '[]'::jsonb,
    robots_directives TEXT[] DEFAULT '{}',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_meta_site ON seo_metadata(site_id);
CREATE INDEX idx_seo_meta_project ON seo_metadata(seo_project_id);
CREATE INDEX idx_seo_meta_post ON seo_metadata(post_id);

CREATE TABLE IF NOT EXISTS seo_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_project_id UUID REFERENCES seo_projects(id) ON DELETE CASCADE,
    post_id UUID,
    total_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    keyword_score NUMERIC(5,2) DEFAULT 0,
    content_score NUMERIC(5,2) DEFAULT 0,
    technical_score NUMERIC(5,2) DEFAULT 0,
    linking_score NUMERIC(5,2) DEFAULT 0,
    readability_score NUMERIC(5,2) DEFAULT 0,
    metadata_score NUMERIC(5,2) DEFAULT 0,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    scored_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seo_scores_site ON seo_scores(site_id);
CREATE INDEX idx_seo_scores_project ON seo_scores(seo_project_id);
CREATE INDEX idx_seo_scores_post ON seo_scores(post_id);
