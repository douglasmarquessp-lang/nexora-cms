-- 000033_add_workflow_publication_id.down.sql

DROP INDEX IF EXISTS idx_wf_jobs_publication;
ALTER TABLE workflow_jobs DROP COLUMN IF EXISTS publication_id;