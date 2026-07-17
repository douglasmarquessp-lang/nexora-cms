CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS autocontent_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id UUID,
    topic VARCHAR(500) NOT NULL,
    title VARCHAR(500) DEFAULT '',
    content_type VARCHAR(100) DEFAULT 'article',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    target_language VARCHAR(10) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    current_step VARCHAR(100) DEFAULT '',
    progress NUMERIC(5,2) DEFAULT 0,
    priority INT NOT NULL DEFAULT 5,
    word_count INT DEFAULT 0,
    tone VARCHAR(100) DEFAULT '',
    audience VARCHAR(100) DEFAULT '',
    keywords TEXT[] DEFAULT '{}',
    style_slug VARCHAR(100) DEFAULT '',
    template_id UUID,
    scheduled_for TIMESTAMPTZ,
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

CREATE INDEX idx_ac_jobs_site ON autocontent_jobs(site_id);
CREATE INDEX idx_ac_jobs_status ON autocontent_jobs(status);
CREATE INDEX idx_ac_jobs_priority ON autocontent_jobs(priority);
CREATE INDEX idx_ac_jobs_language ON autocontent_jobs(language);
CREATE INDEX idx_ac_jobs_topic ON autocontent_jobs(topic);
CREATE INDEX idx_ac_jobs_current_step ON autocontent_jobs(current_step);

CREATE TABLE IF NOT EXISTS autocontent_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    autocontent_job_id UUID NOT NULL REFERENCES autocontent_jobs(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress NUMERIC(5,2) DEFAULT 0,
    depends_on TEXT[] DEFAULT '{}',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT DEFAULT 0,
    error_message TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ac_steps_job ON autocontent_steps(autocontent_job_id);
CREATE INDEX idx_ac_steps_status ON autocontent_steps(status);
CREATE INDEX idx_ac_steps_name ON autocontent_steps(step_name);

CREATE TABLE IF NOT EXISTS autocontent_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    autocontent_job_id UUID NOT NULL REFERENCES autocontent_jobs(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL,
    content TEXT DEFAULT '',
    summary TEXT DEFAULT '',
    score NUMERIC(5,2) DEFAULT 0,
    passed BOOLEAN DEFAULT false,
    data JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ac_results_job ON autocontent_results(autocontent_job_id);
CREATE INDEX idx_ac_results_step ON autocontent_results(step_name);

CREATE TABLE IF NOT EXISTS publication_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    autocontent_job_id UUID REFERENCES autocontent_jobs(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT DEFAULT '',
    excerpt TEXT DEFAULT '',
    language VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 5,
    scheduled_for TIMESTAMPTZ,
    meta_title VARCHAR(200) DEFAULT '',
    meta_description TEXT DEFAULT '',
    slug VARCHAR(500) DEFAULT '',
    featured_image_url TEXT DEFAULT '',
    tags TEXT[] DEFAULT '{}',
    categories TEXT[] DEFAULT '{}',
    published_at TIMESTAMPTZ,
    published_by UUID,
    error_message TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pub_queue_site ON publication_queue(site_id);
CREATE INDEX idx_pub_queue_status ON publication_queue(status);
CREATE INDEX idx_pub_queue_priority ON publication_queue(priority);
CREATE INDEX idx_pub_queue_scheduled ON publication_queue(scheduled_for);
CREATE INDEX idx_pub_queue_job ON publication_queue(autocontent_job_id);

CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT DEFAULT '',
    steps JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_default BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wf_templates_site ON workflow_templates(site_id);
CREATE INDEX idx_wf_templates_default ON workflow_templates(is_default);
