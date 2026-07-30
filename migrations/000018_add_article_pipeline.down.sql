-- 000018_add_article_pipeline.down.sql
-- Rollback article pipeline tables

DROP INDEX IF EXISTS idx_ap_candidates_status;
DROP INDEX IF EXISTS idx_ap_candidates_site;
DROP TABLE IF EXISTS publication_candidates CASCADE;

DROP INDEX IF EXISTS idx_ap_quality_job;
DROP TABLE IF EXISTS article_quality_reports CASCADE;

DROP INDEX IF EXISTS idx_ap_metrics_job;
DROP TABLE IF EXISTS article_pipeline_metrics CASCADE;

DROP INDEX IF EXISTS idx_ap_steps_status;
DROP INDEX IF EXISTS idx_ap_steps_job;
DROP TABLE IF EXISTS article_pipeline_steps CASCADE;

DROP INDEX IF EXISTS idx_ap_jobs_priority;
DROP INDEX IF EXISTS idx_ap_jobs_status;
DROP INDEX IF EXISTS idx_ap_jobs_site;
DROP TABLE IF EXISTS article_pipeline_jobs CASCADE;
