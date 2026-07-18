CREATE TABLE IF NOT EXISTS article_pipeline_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    topic VARCHAR(500) DEFAULT '',
    source_content TEXT DEFAULT '',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    target_language VARCHAR(10) DEFAULT '',
    content_type VARCHAR(100) DEFAULT 'article',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    progress NUMERIC(5,2) DEFAULT 0,
    current_stage VARCHAR(100) DEFAULT '',
    priority INT NOT NULL DEFAULT 5,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    error_message TEXT DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS article_pipeline_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_job_id UUID NOT NULL REFERENCES article_pipeline_jobs(id) ON DELETE CASCADE,
    stage_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress NUMERIC(5,2) DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT DEFAULT 0,
    error_message TEXT DEFAULT '',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    output TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS article_pipeline_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_job_id UUID NOT NULL REFERENCES article_pipeline_jobs(id) ON DELETE CASCADE,
    stage_name VARCHAR(100) NOT NULL DEFAULT '',
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC(10,4) DEFAULT 0,
    metric_unit VARCHAR(50) DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS article_quality_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_job_id UUID NOT NULL REFERENCES article_pipeline_jobs(id) ON DELETE CASCADE,
    stage_name VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    score NUMERIC(5,2) DEFAULT 0,
    checks_passed INT DEFAULT 0,
    checks_failed INT DEFAULT 0,
    checks_total INT DEFAULT 0,
    details JSONB DEFAULT '[]'::jsonb,
    summary TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS publication_candidates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_job_id UUID NOT NULL REFERENCES article_pipeline_jobs(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT DEFAULT '',
    excerpt TEXT DEFAULT '',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    quality_score NUMERIC(5,2) DEFAULT 0,
    seo_score NUMERIC(5,2) DEFAULT 0,
    readability_score NUMERIC(5,2) DEFAULT 0,
    word_count INT DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ap_jobs_site ON article_pipeline_jobs(site_id);
CREATE INDEX idx_ap_jobs_status ON article_pipeline_jobs(status);
CREATE INDEX idx_ap_jobs_priority ON article_pipeline_jobs(priority);
CREATE INDEX idx_ap_steps_job ON article_pipeline_steps(pipeline_job_id);
CREATE INDEX idx_ap_steps_status ON article_pipeline_steps(status);
CREATE INDEX idx_ap_metrics_job ON article_pipeline_metrics(pipeline_job_id);
CREATE INDEX idx_ap_quality_job ON article_quality_reports(pipeline_job_id);
CREATE INDEX idx_ap_candidates_site ON publication_candidates(site_id);
CREATE INDEX idx_ap_candidates_status ON publication_candidates(status);
