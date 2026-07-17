CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS editorial_pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    current_stage VARCHAR(50) NOT NULL DEFAULT 'research',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_editorial_pipelines_job ON editorial_pipelines(article_job_id);
CREATE INDEX idx_editorial_pipelines_site ON editorial_pipelines(site_id);
CREATE INDEX idx_editorial_pipelines_stage ON editorial_pipelines(current_stage);
CREATE INDEX idx_editorial_pipelines_status ON editorial_pipelines(status);

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_id UUID NOT NULL REFERENCES editorial_pipelines(id) ON DELETE CASCADE,
    stage VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    assigned_to UUID,
    notes TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id);
CREATE INDEX idx_pipeline_stages_stage ON pipeline_stages(stage);
CREATE INDEX idx_pipeline_stages_status ON pipeline_stages(status);

CREATE TABLE IF NOT EXISTS editorial_style_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    brand_voice TEXT DEFAULT '',
    tone VARCHAR(100) DEFAULT 'neutral',
    language_level VARCHAR(50) DEFAULT 'standard',
    target_audience TEXT DEFAULT '',
    avg_word_count INT DEFAULT 800,
    heading_structure JSONB DEFAULT '[]'::jsonb,
    prohibited_vocabulary TEXT[] DEFAULT '{}',
    required_expressions TEXT[] DEFAULT '{}',
    personality TEXT DEFAULT '',
    formality_degree VARCHAR(50) DEFAULT 'neutral',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_style_rules_site ON editorial_style_rules(site_id);

CREATE TABLE IF NOT EXISTS editorial_seo_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    primary_keyword TEXT DEFAULT '',
    secondary_keywords TEXT[] DEFAULT '{}',
    long_tail_keywords TEXT[] DEFAULT '{}',
    entities JSONB DEFAULT '[]'::jsonb,
    faq JSONB DEFAULT '[]'::jsonb,
    schema_type VARCHAR(100) DEFAULT '',
    schema_data JSONB DEFAULT '{}'::jsonb,
    meta_title TEXT DEFAULT '',
    meta_description TEXT DEFAULT '',
    slug TEXT DEFAULT '',
    canonical_url TEXT DEFAULT '',
    robots TEXT DEFAULT 'index,follow',
    og_data JSONB DEFAULT '{}'::jsonb,
    twitter_card JSONB DEFAULT '{}'::jsonb,
    alt_text TEXT[] DEFAULT '{}',
    suggested_internal_links TEXT[] DEFAULT '{}',
    suggested_external_links TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_seo_data_job ON editorial_seo_data(article_job_id);
CREATE INDEX idx_seo_data_site ON editorial_seo_data(site_id);

CREATE TABLE IF NOT EXISTS editorial_quality_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    seo_score NUMERIC(5,2) DEFAULT 0,
    readability_score NUMERIC(5,2) DEFAULT 0,
    naturalness_score NUMERIC(5,2) DEFAULT 0,
    eeat_score NUMERIC(5,2) DEFAULT 0,
    keyword_density NUMERIC(5,2) DEFAULT 0,
    heading_structure_score NUMERIC(5,2) DEFAULT 0,
    internal_linking_score NUMERIC(5,2) DEFAULT 0,
    duplicate_detection JSONB DEFAULT '[]'::jsonb,
    repetition_detection JSONB DEFAULT '[]'::jsonb,
    passive_voice_count INT DEFAULT 0,
    avg_sentence_length NUMERIC(5,2) DEFAULT 0,
    paragraph_balance_score NUMERIC(5,2) DEFAULT 0,
    overall_score NUMERIC(5,2) DEFAULT 0,
    report JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quality_scores_job ON editorial_quality_scores(article_job_id);
CREATE INDEX idx_quality_scores_site ON editorial_quality_scores(site_id);

CREATE TABLE IF NOT EXISTS editorial_translations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    source_language VARCHAR(10) NOT NULL,
    target_language VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    translated_slug TEXT DEFAULT '',
    translated_meta JSONB DEFAULT '{}'::jsonb,
    translated_faq JSONB DEFAULT '[]'::jsonb,
    translated_keywords TEXT[] DEFAULT '{}',
    translated_entities JSONB DEFAULT '[]'::jsonb,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_translations_job ON editorial_translations(article_job_id);
CREATE INDEX idx_translations_site ON editorial_translations(site_id);
CREATE INDEX idx_translations_status ON editorial_translations(status);

CREATE TABLE IF NOT EXISTS editorial_prompt_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_job_id UUID NOT NULL REFERENCES article_jobs(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    briefing JSONB DEFAULT '{}'::jsonb,
    style_rules JSONB DEFAULT '{}'::jsonb,
    seo_rules JSONB DEFAULT '{}'::jsonb,
    tone VARCHAR(100) DEFAULT '',
    outline JSONB DEFAULT '[]'::jsonb,
    entities JSONB DEFAULT '[]'::jsonb,
    target_language VARCHAR(10) DEFAULT '',
    audience TEXT DEFAULT '',
    word_count INT DEFAULT 0,
    internal_links TEXT[] DEFAULT '{}',
    constraints TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_prompt_data_job ON editorial_prompt_data(article_job_id);
CREATE INDEX idx_prompt_data_site ON editorial_prompt_data(site_id);
