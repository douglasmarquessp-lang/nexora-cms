-- 000010_add_research_tables.down.sql
-- Rollback research tables

DROP INDEX IF EXISTS idx_research_briefings_job_id;
DROP TABLE IF EXISTS research_briefings CASCADE;

DROP INDEX IF EXISTS idx_research_entities_type;
DROP INDEX IF EXISTS idx_research_entities_job_id;
DROP TABLE IF EXISTS research_entities CASCADE;

DROP INDEX IF EXISTS idx_research_sources_job_id;
DROP TABLE IF EXISTS research_sources CASCADE;

DROP INDEX IF EXISTS idx_research_jobs_language;
DROP INDEX IF EXISTS idx_research_jobs_topic;
DROP INDEX IF EXISTS idx_research_jobs_status;
DROP INDEX IF EXISTS idx_research_jobs_site_id;
DROP TABLE IF EXISTS research_jobs CASCADE;
