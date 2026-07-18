DROP TABLE IF EXISTS seo_improvements CASCADE;

ALTER TABLE seo_scores
    DROP COLUMN IF EXISTS multilingual_score,
    DROP COLUMN IF EXISTS heading_score,
    DROP COLUMN IF EXISTS slug_score,
    DROP COLUMN IF EXISTS image_score,
    DROP COLUMN IF EXISTS schema_score,
    DROP COLUMN IF EXISTS topical_authority_score,
    DROP COLUMN IF EXISTS freshness_score,
    DROP COLUMN IF EXISTS eeat_score;

ALTER TABLE seo_audits
    DROP COLUMN IF EXISTS freshness_issues,
    DROP COLUMN IF EXISTS eeat_issues,
    DROP COLUMN IF EXISTS checklist_items,
    DROP COLUMN IF EXISTS link_suggestions,
    DROP COLUMN IF EXISTS content_gap_detected,
    DROP COLUMN IF EXISTS cannibalization_detected,
    DROP COLUMN IF EXISTS orphan_detected,
    DROP COLUMN IF EXISTS meta_issues,
    DROP COLUMN IF EXISTS title_issues,
    DROP COLUMN IF EXISTS slug_issues,
    DROP COLUMN IF EXISTS slug_score,
    DROP COLUMN IF EXISTS schema_issues,
    DROP COLUMN IF EXISTS image_alt_issues,
    DROP COLUMN IF EXISTS heading_issues,
    DROP COLUMN IF EXISTS freshness_score,
    DROP COLUMN IF EXISTS eeat_score;

ALTER TABLE seo_clusters
    DROP COLUMN IF EXISTS parent_cluster_id,
    DROP COLUMN IF EXISTS content_gap_articles,
    DROP COLUMN IF EXISTS internal_links_count,
    DROP COLUMN IF EXISTS semantic_entities,
    DROP COLUMN IF EXISTS topical_authority_score;

ALTER TABLE seo_keywords
    DROP COLUMN IF EXISTS topical_relevance,
    DROP COLUMN IF EXISTS semantic_entities,
    DROP COLUMN IF EXISTS content_gap_score,
    DROP COLUMN IF EXISTS cannibalization_score,
    DROP COLUMN IF EXISTS cluster_id;

ALTER TABLE seo_projects
    DROP COLUMN IF EXISTS content_type,
    DROP COLUMN IF EXISTS ai_suggestions,
    DROP COLUMN IF EXISTS checklist,
    DROP COLUMN IF EXISTS meta_description_target,
    DROP COLUMN IF EXISTS meta_title_target,
    DROP COLUMN IF EXISTS slug_target,
    DROP COLUMN IF EXISTS topical_authority_score,
    DROP COLUMN IF EXISTS freshness_score,
    DROP COLUMN IF EXISTS eeat_score;
