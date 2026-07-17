CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS generation_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    article_job_id UUID REFERENCES article_jobs(id) ON DELETE SET NULL,
    research_job_id UUID REFERENCES research_jobs(id) ON DELETE SET NULL,
    priority INT NOT NULL DEFAULT 5,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    category VARCHAR(100) DEFAULT '',
    article_type VARCHAR(100) DEFAULT 'article',
    expected_size VARCHAR(50) DEFAULT 'medium',
    style_slug VARCHAR(100) DEFAULT '',
    keywords TEXT[] DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress NUMERIC(5,2) DEFAULT 0,
    current_stage VARCHAR(100) DEFAULT '',
    error_message TEXT DEFAULT '',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gen_jobs_site ON generation_jobs(site_id);
CREATE INDEX idx_gen_jobs_status ON generation_jobs(status);
CREATE INDEX idx_gen_jobs_priority ON generation_jobs(priority);
CREATE INDEX idx_gen_jobs_language ON generation_jobs(language);
CREATE INDEX idx_gen_jobs_article_job ON generation_jobs(article_job_id);

CREATE TABLE IF NOT EXISTS generation_pipeline (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_job_id UUID NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
    stage VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress NUMERIC(5,2) DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT DEFAULT 0,
    error_message TEXT DEFAULT '',
    retry_count INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gen_pipeline_job ON generation_pipeline(generation_job_id);
CREATE INDEX idx_gen_pipeline_stage ON generation_pipeline(stage);
CREATE INDEX idx_gen_pipeline_status ON generation_pipeline(status);

CREATE TABLE IF NOT EXISTS generation_pipeline_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_job_id UUID NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
    stage VARCHAR(100) DEFAULT '',
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL DEFAULT '',
    details JSONB DEFAULT '{}'::jsonb,
    duration_ms BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gen_logs_job ON generation_pipeline_logs(generation_job_id);
CREATE INDEX idx_gen_logs_stage ON generation_pipeline_logs(stage);
CREATE INDEX idx_gen_logs_level ON generation_pipeline_logs(level);
CREATE INDEX idx_gen_logs_created ON generation_pipeline_logs(created_at);

CREATE TABLE IF NOT EXISTS generation_quality_gates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_job_id UUID NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
    stage VARCHAR(100) NOT NULL DEFAULT 'final',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    seo_score NUMERIC(5,2) DEFAULT 0,
    readability_score NUMERIC(5,2) DEFAULT 0,
    eeat_score NUMERIC(5,2) DEFAULT 0,
    keyword_density NUMERIC(5,2) DEFAULT 0,
    heading_score NUMERIC(5,2) DEFAULT 0,
    internal_linking_score NUMERIC(5,2) DEFAULT 0,
    required_content_passed BOOLEAN DEFAULT false,
    min_size_passed BOOLEAN DEFAULT false,
    metadata_passed BOOLEAN DEFAULT false,
    overall_passed BOOLEAN DEFAULT false,
    report JSONB DEFAULT '{}'::jsonb,
    checked_by UUID,
    checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gen_quality_job ON generation_quality_gates(generation_job_id);
CREATE INDEX idx_gen_quality_stage ON generation_quality_gates(stage);

CREATE TABLE IF NOT EXISTS generation_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_jobs INT NOT NULL DEFAULT 0,
    completed_jobs INT NOT NULL DEFAULT 0,
    failed_jobs INT NOT NULL DEFAULT 0,
    cancelled_jobs INT NOT NULL DEFAULT 0,
    avg_duration_ms BIGINT DEFAULT 0,
    avg_success_rate NUMERIC(5,2) DEFAULT 0,
    total_errors INT NOT NULL DEFAULT 0,
    throughput NUMERIC(10,2) DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_gen_stats_site_date ON generation_stats(site_id, date);
CREATE INDEX idx_gen_stats_date ON generation_stats(date);
