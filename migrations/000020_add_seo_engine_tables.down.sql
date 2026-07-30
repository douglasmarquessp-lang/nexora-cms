-- 000020_add_seo_engine_tables.down.sql
-- Rollback SEO engine enhancements: new columns + seo_improvements table

DROP INDEX IF EXISTS idx_seo_kw_cluster;
DROP INDEX IF EXISTS idx_seo_improvements_priority;
DROP INDEX IF EXISTS idx_seo_improvements_status;
DROP INDEX IF EXISTS idx_seo_improvements_category;
DROP INDEX IF EXISTS idx_seo_improvements_project;
DROP INDEX IF EXISTS idx_seo_improvements_site;
DROP TABLE IF EXISTS seo_improvements CASCADE;

-- Revert ALTER TABLE seo_scores
ALTER TABLE seo_scores DROP COLUMN IF EXISTS heading_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS multilingual_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS slug_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS image_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS schema_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS topical_authority_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS freshness_score;
ALTER TABLE seo_scores DROP COLUMN IF EXISTS eeat_score;

-- Revert ALTER TABLE seo_audits
ALTER TABLE seo_audits DROP COLUMN IF EXISTS freshness_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS eeat_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS checklist_items;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS link_suggestions;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS content_gap_detected;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS cannibalization_detected;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS orphan_detected;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS meta_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS title_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS slug_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS slug_score;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS schema_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS image_alt_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS heading_issues;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS freshness_score;
ALTER TABLE seo_audits DROP COLUMN IF EXISTS eeat_score;

-- Revert ALTER TABLE seo_clusters
ALTER TABLE seo_clusters DROP CONSTRAINT IF EXISTS seo_clusters_parent_cluster_id_fkey;
ALTER TABLE seo_clusters DROP COLUMN IF EXISTS parent_cluster_id;
ALTER TABLE seo_clusters DROP COLUMN IF EXISTS content_gap_articles;
ALTER TABLE seo_clusters DROP COLUMN IF EXISTS internal_links_count;
ALTER TABLE seo_clusters DROP COLUMN IF EXISTS semantic_entities;
ALTER TABLE seo_clusters DROP COLUMN IF EXISTS topical_authority_score;

-- Revert ALTER TABLE seo_keywords
ALTER TABLE seo_keywords DROP COLUMN IF EXISTS topical_relevance;
ALTER TABLE seo_keywords DROP COLUMN IF EXISTS semantic_entities;
ALTER TABLE seo_keywords DROP COLUMN IF EXISTS content_gap_score;
ALTER TABLE seo_keywords DROP COLUMN IF EXISTS cannibalization_score;
ALTER TABLE seo_keywords DROP COLUMN IF EXISTS cluster_id;

-- Revert ALTER TABLE seo_projects
ALTER TABLE seo_projects DROP COLUMN IF EXISTS content_type;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS ai_suggestions;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS checklist;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS meta_description_target;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS meta_title_target;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS slug_target;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS topical_authority_score;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS freshness_score;
ALTER TABLE seo_projects DROP COLUMN IF EXISTS eeat_score;
