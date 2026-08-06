-- 000026_add_post_seo_columns.down.sql

DROP INDEX IF EXISTS idx_posts_site_seo_score;

ALTER TABLE posts DROP COLUMN IF EXISTS seo_issues;
ALTER TABLE posts DROP COLUMN IF EXISTS seo_analyzed_at;
ALTER TABLE posts DROP COLUMN IF EXISTS seo_score;
