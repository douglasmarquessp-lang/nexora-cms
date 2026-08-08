-- 000035_add_workflow_logs.down.sql

DROP POLICY IF EXISTS workflow_logs_isolation ON workflow_logs;

ALTER TABLE workflow_logs DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_wflogs_created;
DROP INDEX IF EXISTS idx_wflogs_level;
DROP INDEX IF EXISTS idx_wflogs_site;
DROP INDEX IF EXISTS idx_wflogs_job;

DROP TABLE IF EXISTS workflow_logs CASCADE;