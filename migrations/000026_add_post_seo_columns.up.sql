-- 000026_add_post_seo_columns.up.sql
-- Store the latest SEO audit results on the post itself so the publish gate
-- and listings can consult scores without joining seo_audits.

ALTER TABLE posts ADD COLUMN IF NOT EXISTS seo_score NUMERIC(5,2) DEFAULT 0;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS seo_analyzed_at TIMESTAMPTZ;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS seo_issues JSONB DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_posts_site_seo_score ON posts(site_id, seo_score DESC) WHERE deleted_at IS NULL;
