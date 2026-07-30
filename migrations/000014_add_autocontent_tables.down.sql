-- 000014_add_autocontent_tables.down.sql
-- Rollback autocontent tables.
--
-- NOTE: The original 000014 created a table named 'publication_queue'
-- which caused a naming conflict with 000019. In Sprint 3.7 the table
-- was renamed to 'autocontent_queue' in this migration. The down
-- migration drops 'autocontent_queue' (the renamed table), not
-- 'publication_queue' (which is owned by 000019).

DROP INDEX IF EXISTS idx_wf_templates_default;
DROP INDEX IF EXISTS idx_wf_templates_site;
DROP TABLE IF EXISTS workflow_templates CASCADE;

DROP INDEX IF EXISTS idx_ac_queue_job;
DROP INDEX IF EXISTS idx_ac_queue_scheduled;
DROP INDEX IF EXISTS idx_ac_queue_priority;
DROP INDEX IF EXISTS idx_ac_queue_status;
DROP INDEX IF EXISTS idx_ac_queue_site;
DROP TABLE IF EXISTS autocontent_queue CASCADE;

DROP INDEX IF EXISTS idx_ac_results_step;
DROP INDEX IF EXISTS idx_ac_results_job;
DROP TABLE IF EXISTS autocontent_results CASCADE;

DROP INDEX IF EXISTS idx_ac_steps_name;
DROP INDEX IF EXISTS idx_ac_steps_status;
DROP INDEX IF EXISTS idx_ac_steps_job;
DROP TABLE IF EXISTS autocontent_steps CASCADE;

DROP INDEX IF EXISTS idx_ac_jobs_current_step;
DROP INDEX IF EXISTS idx_ac_jobs_topic;
DROP INDEX IF EXISTS idx_ac_jobs_language;
DROP INDEX IF EXISTS idx_ac_jobs_priority;
DROP INDEX IF EXISTS idx_ac_jobs_status;
DROP INDEX IF EXISTS idx_ac_jobs_site;
DROP TABLE IF EXISTS autocontent_jobs CASCADE;
