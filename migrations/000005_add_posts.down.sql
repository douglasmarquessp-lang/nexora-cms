-- 000005_add_posts.down.sql
-- Rollback posts, categories, tags and relations

DROP TRIGGER IF EXISTS set_posts_updated_at ON posts;
DROP TRIGGER IF EXISTS set_categories_updated_at ON categories;
DROP TRIGGER IF EXISTS set_tags_updated_at ON tags;

DROP POLICY IF EXISTS tags_isolation ON tags;
DROP POLICY IF EXISTS categories_isolation ON categories;
DROP POLICY IF EXISTS posts_isolation ON posts;

ALTER TABLE tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE categories DISABLE ROW LEVEL SECURITY;
ALTER TABLE posts DISABLE ROW LEVEL SECURITY;

-- Junction tables first (FK targets)
DROP INDEX IF EXISTS idx_post_tags_tag;
DROP TABLE IF EXISTS post_tags CASCADE;

DROP INDEX IF EXISTS idx_post_categories_category;
DROP TABLE IF EXISTS post_categories CASCADE;

-- Main tables
DROP INDEX IF EXISTS idx_tags_site;
DROP TABLE IF EXISTS tags CASCADE;

DROP INDEX IF EXISTS idx_categories_sort;
DROP INDEX IF EXISTS idx_categories_parent;
DROP INDEX IF EXISTS idx_categories_site;
DROP TABLE IF EXISTS categories CASCADE;

DROP INDEX IF EXISTS idx_posts_search;
DROP INDEX IF EXISTS idx_posts_created;
DROP INDEX IF EXISTS idx_posts_published;
DROP INDEX IF EXISTS idx_posts_author;
DROP INDEX IF EXISTS idx_posts_site_slug;
DROP INDEX IF EXISTS idx_posts_site_status;
DROP TABLE IF EXISTS posts CASCADE;
