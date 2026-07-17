CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS writing_styles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    config JSONB DEFAULT '{}'::jsonb,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_writing_styles_site_slug ON writing_styles(site_id, slug);

CREATE TABLE IF NOT EXISTS article_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    research_job_id UUID REFERENCES research_jobs(id) ON DELETE SET NULL,
    style_id UUID REFERENCES writing_styles(id) ON DELETE SET NULL,
    style_name VARCHAR(100) NOT NULL DEFAULT '',
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    headline TEXT DEFAULT '',
    seo_title TEXT DEFAULT '',
    slug TEXT DEFAULT '',
    meta_description TEXT DEFAULT '',
    target_audience TEXT DEFAULT '',
    tone VARCHAR(100) DEFAULT 'neutral',
    formality VARCHAR(50) DEFAULT 'neutral',
    seo_goal TEXT DEFAULT '',
    desired_size VARCHAR(50) DEFAULT 'medium',
    created_by UUID,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_article_jobs_site_id ON article_jobs(site_id);
CREATE INDEX idx_article_jobs_status ON article_jobs(status);
CREATE INDEX idx_article_jobs_research_job_id ON article_jobs(research_job_id);
CREATE INDEX idx_article_jobs_language ON article_jobs(language);

CREATE TABLE IF NOT EXISTS article_outlines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    section_type VARCHAR(50) NOT NULL DEFAULT 'h2',
    title VARCHAR(500) NOT NULL,
    level INT NOT NULL DEFAULT 0,
    content TEXT DEFAULT '',
    position INT NOT NULL DEFAULT 0,
    word_count_target INT DEFAULT 0,
    keywords TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_article_outlines_job_id ON article_outlines(article_job_id);

CREATE TABLE IF NOT EXISTS article_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    outline_id UUID REFERENCES article_outlines(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL DEFAULT '',
    content TEXT DEFAULT '',
    word_count INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_article_sections_job_id ON article_sections(article_job_id);
CREATE INDEX idx_article_sections_status ON article_sections(status);

CREATE TABLE IF NOT EXISTS article_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    version INT NOT NULL,
    headline TEXT DEFAULT '',
    seo_title TEXT DEFAULT '',
    slug TEXT DEFAULT '',
    meta_description TEXT DEFAULT '',
    sections JSONB DEFAULT '[]'::jsonb,
    content JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    summary TEXT DEFAULT '',
    change_log TEXT DEFAULT '',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_article_versions_job_version ON article_versions(article_job_id, version);
CREATE INDEX idx_article_versions_job_id ON article_versions(article_job_id);

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Journalistic', 'journalistic',
       'Fact-based reporting with inverted pyramid structure',
       '{"tone":"neutral","formality":"formal","audience":"general","typical_length":800,"seo_focus":"headline"}',
       true FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Technical', 'technical',
       'Detailed technical documentation and guides',
       '{"tone":"objective","formality":"formal","audience":"developers","typical_length":1500,"seo_focus":"keywords"}',
       false FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Tutorial', 'tutorial',
       'Step-by-step instructional content',
       '{"tone":"friendly","formality":"semi-formal","audience":"beginners","typical_length":1200,"seo_focus":"how_to"}',
       false FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Review', 'review',
       'Product or service evaluation and critique',
       '{"tone":"analytical","formality":"semi-formal","audience":"buyers","typical_length":1000,"seo_focus":"comparison"}',
       false FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Comparative', 'comparative',
       'Side-by-side comparison of options',
       '{"tone":"objective","formality":"neutral","audience":"decision_makers","typical_length":1200,"seo_focus":"vs_keywords"}',
       false FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'List', 'list',
       'Curated list-based article format',
       '{"tone":"engaging","formality":"casual","audience":"general","typical_length":800,"seo_focus":"numbered_lists"}',
       false FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Opinion', 'opinion',
       'Personal perspective and analysis',
       '{"tone":"persuasive","formality":"semi-formal","audience":"engaged","typical_length":700,"seo_focus":"opinion_keywords"}',
       false FROM sites s;

INSERT INTO writing_styles (site_id, name, slug, description, config, is_default)
SELECT s.id, 'Complete Guide', 'complete_guide',
       'Comprehensive all-in-one guide on a subject',
       '{"tone":"authoritative","formality":"formal","audience":"learners","typical_length":3000,"seo_focus":"comprehensive"}',
       false FROM sites s;
