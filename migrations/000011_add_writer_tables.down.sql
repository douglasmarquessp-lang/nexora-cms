-- 000011_add_writer_tables.down.sql
-- Rollback writer/article tables and seed data

-- Seed data (delete the 8 auto-inserted writing styles per site)
DELETE FROM writing_styles WHERE slug IN (
    'journalistic', 'technical', 'tutorial', 'review',
    'comparative', 'list', 'opinion', 'complete_guide'
);

DROP INDEX IF EXISTS idx_article_versions_job_id;
DROP INDEX IF EXISTS idx_article_versions_job_version;
DROP TABLE IF EXISTS article_versions CASCADE;

DROP INDEX IF EXISTS idx_article_sections_status;
DROP INDEX IF EXISTS idx_article_sections_job_id;
DROP TABLE IF EXISTS article_sections CASCADE;

DROP INDEX IF EXISTS idx_article_outlines_job_id;
DROP TABLE IF EXISTS article_outlines CASCADE;

DROP INDEX IF EXISTS idx_article_jobs_language;
DROP INDEX IF EXISTS idx_article_jobs_research_job_id;
DROP INDEX IF EXISTS idx_article_jobs_status;
DROP INDEX IF EXISTS idx_article_jobs_site_id;
DROP TABLE IF EXISTS article_jobs CASCADE;

DROP INDEX IF EXISTS idx_writing_styles_site_slug;
DROP TABLE IF EXISTS writing_styles CASCADE;
