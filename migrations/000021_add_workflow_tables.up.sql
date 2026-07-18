CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS workflow_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL DEFAULT '',
    content_type VARCHAR(50) NOT NULL DEFAULT 'article',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    target_language VARCHAR(10) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    current_step VARCHAR(100) DEFAULT '',
    progress NUMERIC(5,2) DEFAULT 0,
    priority INT NOT NULL DEFAULT 5,
    word_count INT DEFAULT 0,
    tone VARCHAR(100) DEFAULT '',
    audience VARCHAR(200) DEFAULT '',
    keywords TEXT[] DEFAULT '{}',
    style_slug VARCHAR(100) DEFAULT '',
    source_job_id UUID,
    scheduled_for TIMESTAMPTZ,
    error_message TEXT DEFAULT '',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    generate_pt BOOLEAN DEFAULT false,
    generate_en BOOLEAN DEFAULT false,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_job_id UUID NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress NUMERIC(5,2) DEFAULT 0,
    depends_on TEXT[] DEFAULT '{}',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT DEFAULT 0,
    error_message TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    workflow_job_id UUID REFERENCES workflow_jobs(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT DEFAULT '',
    excerpt TEXT DEFAULT '',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 5,
    scheduled_for TIMESTAMPTZ,
    is_paused BOOLEAN DEFAULT false,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    meta_title VARCHAR(200) DEFAULT '',
    meta_description TEXT DEFAULT '',
    slug VARCHAR(500) DEFAULT '',
    featured_image_url VARCHAR(1000) DEFAULT '',
    tags TEXT[] DEFAULT '{}',
    categories TEXT[] DEFAULT '{}',
    published_at TIMESTAMPTZ,
    published_by UUID,
    error_message TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    workflow_job_id UUID REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    queue_id UUID REFERENCES workflow_queue(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL DEFAULT 'job',
    entity_id UUID,
    previous_status VARCHAR(50) DEFAULT '',
    new_status VARCHAR(50) DEFAULT '',
    details JSONB DEFAULT '{}'::jsonb,
    error_message TEXT DEFAULT '',
    user_id UUID,
    duration_ms BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    workflow_job_id UUID REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    queue_id UUID REFERENCES workflow_queue(id) ON DELETE SET NULL,
    notification_type VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL DEFAULT '',
    message TEXT DEFAULT '',
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    read BOOLEAN DEFAULT false,
    action_url VARCHAR(1000) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_dashboard (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    total_jobs BIGINT NOT NULL DEFAULT 0,
    running_jobs BIGINT NOT NULL DEFAULT 0,
    completed_jobs BIGINT NOT NULL DEFAULT 0,
    failed_jobs BIGINT NOT NULL DEFAULT 0,
    paused_jobs BIGINT NOT NULL DEFAULT 0,
    queue_size BIGINT NOT NULL DEFAULT 0,
    stalled_queue BIGINT NOT NULL DEFAULT 0,
    pending_review BIGINT NOT NULL DEFAULT 0,
    scheduled_publications BIGINT NOT NULL DEFAULT 0,
    recent_publications BIGINT NOT NULL DEFAULT 0,
    avg_execution_ms NUMERIC(12,2) DEFAULT 0,
    success_rate NUMERIC(5,2) DEFAULT 0,
    failure_rate NUMERIC(5,2) DEFAULT 0,
    throughput_hourly NUMERIC(10,2) DEFAULT 0,
    worker_utilization NUMERIC(5,2) DEFAULT 0,
    data JSONB DEFAULT '{}'::jsonb,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wf_jobs_site ON workflow_jobs(site_id);
CREATE INDEX IF NOT EXISTS idx_wf_jobs_status ON workflow_jobs(status);
CREATE INDEX IF NOT EXISTS idx_wf_jobs_user ON workflow_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_wf_jobs_priority ON workflow_jobs(priority);
CREATE INDEX IF NOT EXISTS idx_wf_jobs_language ON workflow_jobs(language);
CREATE INDEX IF NOT EXISTS idx_wf_jobs_scheduled ON workflow_jobs(scheduled_for);
CREATE INDEX IF NOT EXISTS idx_wf_jobs_created ON workflow_jobs(created_at);

CREATE INDEX IF NOT EXISTS idx_wf_steps_job ON workflow_steps(workflow_job_id);
CREATE INDEX IF NOT EXISTS idx_wf_steps_name ON workflow_steps(step_name);
CREATE INDEX IF NOT EXISTS idx_wf_steps_status ON workflow_steps(status);

CREATE INDEX IF NOT EXISTS idx_wf_queue_site ON workflow_queue(site_id);
CREATE INDEX IF NOT EXISTS idx_wf_queue_status ON workflow_queue(status);
CREATE INDEX IF NOT EXISTS idx_wf_queue_priority ON workflow_queue(priority);
CREATE INDEX IF NOT EXISTS idx_wf_queue_scheduled ON workflow_queue(scheduled_for);
CREATE INDEX IF NOT EXISTS idx_wf_queue_paused ON workflow_queue(is_paused);

CREATE INDEX IF NOT EXISTS idx_wf_history_site ON workflow_history(site_id);
CREATE INDEX IF NOT EXISTS idx_wf_history_job ON workflow_history(workflow_job_id);
CREATE INDEX IF NOT EXISTS idx_wf_history_action ON workflow_history(action);
CREATE INDEX IF NOT EXISTS idx_wf_history_created ON workflow_history(created_at);

CREATE INDEX IF NOT EXISTS idx_wf_notifications_site ON workflow_notifications(site_id);
CREATE INDEX IF NOT EXISTS idx_wf_notifications_type ON workflow_notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_wf_notifications_read ON workflow_notifications(read);
CREATE INDEX IF NOT EXISTS idx_wf_notifications_created ON workflow_notifications(created_at);

CREATE INDEX IF NOT EXISTS idx_wf_dashboard_site ON workflow_dashboard(site_id);
CREATE INDEX IF NOT EXISTS idx_wf_dashboard_snapshot ON workflow_dashboard(snapshot_at);
