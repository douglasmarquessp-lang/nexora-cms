-- 000015_add_seo_tables.down.sql
-- Rollback SEO tables

DROP INDEX IF EXISTS idx_seo_scores_post;
DROP INDEX IF EXISTS idx_seo_scores_project;
DROP INDEX IF EXISTS idx_seo_scores_site;
DROP TABLE IF EXISTS seo_scores CASCADE;

DROP INDEX IF EXISTS idx_seo_meta_post;
DROP INDEX IF EXISTS idx_seo_meta_project;
DROP INDEX IF EXISTS idx_seo_meta_site;
DROP TABLE IF EXISTS seo_metadata CASCADE;

DROP INDEX IF EXISTS idx_seo_links_source;
DROP INDEX IF EXISTS idx_seo_links_project;
DROP INDEX IF EXISTS idx_seo_links_site;
DROP TABLE IF EXISTS seo_internal_links CASCADE;

DROP INDEX IF EXISTS idx_seo_audits_post;
DROP INDEX IF EXISTS idx_seo_audits_project;
DROP INDEX IF EXISTS idx_seo_audits_site;
DROP TABLE IF EXISTS seo_audits CASCADE;

DROP INDEX IF EXISTS idx_seo_clusters_site;
DROP TABLE IF EXISTS seo_clusters CASCADE;

DROP INDEX IF EXISTS idx_seo_kw_intent;
DROP INDEX IF EXISTS idx_seo_kw_type;
DROP INDEX IF EXISTS idx_seo_kw_project;
DROP INDEX IF EXISTS idx_seo_kw_site;
DROP TABLE IF EXISTS seo_keywords CASCADE;

DROP INDEX IF EXISTS idx_seo_projects_post;
DROP INDEX IF EXISTS idx_seo_projects_status;
DROP INDEX IF EXISTS idx_seo_projects_site;
DROP TABLE IF EXISTS seo_projects CASCADE;
