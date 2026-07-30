-- 000021_add_workflow_tables.down.sql
-- Rollback workflow management tables

DROP INDEX IF EXISTS idx_wf_dashboard_snapshot;
DROP INDEX IF EXISTS idx_wf_dashboard_site;
DROP TABLE IF EXISTS workflow_dashboard CASCADE;

DROP INDEX IF EXISTS idx_wf_notifications_created;
DROP INDEX IF EXISTS idx_wf_notifications_read;
DROP INDEX IF EXISTS idx_wf_notifications_type;
DROP INDEX IF EXISTS idx_wf_notifications_site;
DROP TABLE IF EXISTS workflow_notifications CASCADE;

DROP INDEX IF EXISTS idx_wf_history_created;
DROP INDEX IF EXISTS idx_wf_history_action;
DROP INDEX IF EXISTS idx_wf_history_job;
DROP INDEX IF EXISTS idx_wf_history_site;
DROP TABLE IF EXISTS workflow_history CASCADE;

DROP INDEX IF EXISTS idx_wf_queue_paused;
DROP INDEX IF EXISTS idx_wf_queue_scheduled;
DROP INDEX IF EXISTS idx_wf_queue_priority;
DROP INDEX IF EXISTS idx_wf_queue_status;
DROP INDEX IF EXISTS idx_wf_queue_site;
DROP TABLE IF EXISTS workflow_queue CASCADE;

DROP INDEX IF EXISTS idx_wf_steps_status;
DROP INDEX IF EXISTS idx_wf_steps_name;
DROP INDEX IF EXISTS idx_wf_steps_job;
DROP TABLE IF EXISTS workflow_steps CASCADE;

DROP INDEX IF EXISTS idx_wf_jobs_created;
DROP INDEX IF EXISTS idx_wf_jobs_scheduled;
DROP INDEX IF EXISTS idx_wf_jobs_language;
DROP INDEX IF EXISTS idx_wf_jobs_priority;
DROP INDEX IF EXISTS idx_wf_jobs_user;
DROP INDEX IF EXISTS idx_wf_jobs_status;
DROP INDEX IF EXISTS idx_wf_jobs_site;
DROP TABLE IF EXISTS workflow_jobs CASCADE;
