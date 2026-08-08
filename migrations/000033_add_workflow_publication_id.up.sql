-- 000033_add_workflow_publication_id.up.sql
-- Track the publication created by an automated workflow job so the UI can
-- link from a completed job to its published article.

ALTER TABLE workflow_jobs ADD COLUMN IF NOT EXISTS publication_id UUID REFERENCES publications(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_wf_jobs_publication ON workflow_jobs(publication_id);