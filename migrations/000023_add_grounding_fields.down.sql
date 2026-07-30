-- 000023_add_grounding_fields.down.sql
-- Rollback grounding-related fields and article_sources table

-- Drop article_sources table and its indexes
DROP INDEX IF EXISTS idx_article_sources_freshness;
DROP INDEX IF EXISTS idx_article_sources_verified;
DROP INDEX IF EXISTS idx_article_sources_url;
DROP INDEX IF EXISTS idx_article_sources_workflow;
DROP INDEX IF EXISTS idx_article_sources_pipeline;
DROP INDEX IF EXISTS idx_article_sources_article;
DROP INDEX IF EXISTS idx_article_sources_site;
DROP TABLE IF EXISTS article_sources CASCADE;

-- Revert ALTER TABLE research_sources
ALTER TABLE research_sources DROP COLUMN IF EXISTS grounding_metadata;
ALTER TABLE research_sources DROP COLUMN IF EXISTS retrieved_at;
ALTER TABLE research_sources DROP COLUMN IF EXISTS is_verified;
ALTER TABLE research_sources DROP COLUMN IF EXISTS freshness_score;
