-- 000036: workflow job review lifecycle + version history
-- Adds the editorial review columns to workflow_jobs and a version table
-- that snapshots each article draft (regenerate never deletes prior content).

ALTER TABLE workflow_jobs
    ADD COLUMN review_status   TEXT        NOT NULL DEFAULT 'generated',
    ADD COLUMN approved_by     UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN approved_at     TIMESTAMPTZ,
    ADD COLUMN rejected_by     UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN rejected_at     TIMESTAMPTZ,
    ADD COLUMN rejection_reason TEXT,
    ADD COLUMN revision        INTEGER     NOT NULL DEFAULT 1;

CREATE INDEX idx_wf_jobs_review_status ON workflow_jobs (site_id, review_status);

CREATE TABLE workflow_job_versions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id            UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    workflow_job_id    UUID NOT NULL REFERENCES workflow_jobs(id) ON DELETE CASCADE,
    version            INTEGER NOT NULL,
    title              TEXT NOT NULL DEFAULT '',
    slug               TEXT NOT NULL DEFAULT '',
    content            TEXT NOT NULL DEFAULT '',
    meta_title         TEXT NOT NULL DEFAULT '',
    meta_description   TEXT NOT NULL DEFAULT '',
    keyword            TEXT NOT NULL DEFAULT '',
    featured_image_url TEXT NOT NULL DEFAULT '',
    featured_image_alt TEXT NOT NULL DEFAULT '',
    language           TEXT NOT NULL DEFAULT 'en',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workflow_job_id, version)
);

CREATE INDEX idx_wf_job_versions_job ON workflow_job_versions (workflow_job_id, version);
CREATE INDEX idx_wf_job_versions_site ON workflow_job_versions (site_id);

ALTER TABLE workflow_job_versions ENABLE ROW LEVEL SECURITY;

CREATE POLICY workflow_job_versions_isolation ON workflow_job_versions
    USING (site_id = current_setting('app.current_site_id')::UUID);
