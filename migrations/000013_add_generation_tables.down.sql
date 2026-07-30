-- 000013_add_generation_tables.down.sql
-- Rollback content generation orchestrator tables

DROP INDEX IF EXISTS idx_gen_stats_date;
DROP INDEX IF EXISTS idx_gen_stats_site_date;
DROP TABLE IF EXISTS generation_stats CASCADE;

DROP INDEX IF EXISTS idx_gen_quality_stage;
DROP INDEX IF EXISTS idx_gen_quality_job;
DROP TABLE IF EXISTS generation_quality_gates CASCADE;

DROP INDEX IF EXISTS idx_gen_logs_created;
DROP INDEX IF EXISTS idx_gen_logs_level;
DROP INDEX IF EXISTS idx_gen_logs_stage;
DROP INDEX IF EXISTS idx_gen_logs_job;
DROP TABLE IF EXISTS generation_pipeline_logs CASCADE;

DROP INDEX IF EXISTS idx_gen_pipeline_status;
DROP INDEX IF EXISTS idx_gen_pipeline_stage;
DROP INDEX IF EXISTS idx_gen_pipeline_job;
DROP TABLE IF EXISTS generation_pipeline CASCADE;

DROP INDEX IF EXISTS idx_gen_jobs_article_job;
DROP INDEX IF EXISTS idx_gen_jobs_language;
DROP INDEX IF EXISTS idx_gen_jobs_priority;
DROP INDEX IF EXISTS idx_gen_jobs_status;
DROP INDEX IF EXISTS idx_gen_jobs_site;
DROP TABLE IF EXISTS generation_jobs CASCADE;
