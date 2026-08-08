-- 000035_add_workflow_logs.up.sql
-- Dedicated workflow log table.
--
-- The workflow module previously wrote its logs into
-- generation_pipeline_logs (migration 000013), whose FK
-- generation_job_id -> generation_jobs(id) belongs to the
-- contentgenerator module. Workflow jobs are not generation_jobs,
-- so every workflow log insert failed with SQLSTATE 23503
-- (foreign key violation) and the log was lost.
-- workflow_logs owns its own FK -> workflow_jobs(id) and follows
-- the workflow module RLS/site_id pattern (see workflow_history).

CREATE TABLE IF NOT EXISTS workflow_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    workflow_job_id UUID NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    step VARCHAR(100) DEFAULT '',
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL DEFAULT '',
    details JSONB DEFAULT '{}'::jsonb,
    duration_ms BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wflogs_job ON workflow_logs(workflow_job_id);
CREATE INDEX IF NOT EXISTS idx_wflogs_site ON workflow_logs(site_id);
CREATE INDEX IF NOT EXISTS idx_wflogs_level ON workflow_logs(level);
CREATE INDEX IF NOT EXISTS idx_wflogs_created ON workflow_logs(created_at);

ALTER TABLE workflow_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY workflow_logs_isolation ON workflow_logs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );