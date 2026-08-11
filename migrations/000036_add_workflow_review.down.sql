-- 000036: reverse — workflow job review lifecycle + version history

DROP POLICY IF EXISTS workflow_job_versions_isolation ON workflow_job_versions;
ALTER TABLE workflow_job_versions DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_wf_job_versions_site;
DROP INDEX IF EXISTS idx_wf_job_versions_job;
DROP TABLE IF EXISTS workflow_job_versions;

DROP INDEX IF EXISTS idx_wf_jobs_review_status;
ALTER TABLE workflow_jobs
    DROP COLUMN IF EXISTS revision,
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS rejected_at,
    DROP COLUMN IF EXISTS rejected_by,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS review_status;
