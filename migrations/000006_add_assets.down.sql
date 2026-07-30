-- 000006_add_assets.down.sql
-- Rollback assets, post_assets and autosave

DROP TRIGGER IF EXISTS set_assets_updated_at ON assets;

DROP POLICY IF EXISTS post_autosaves_isolation ON post_autosaves;
DROP POLICY IF EXISTS post_assets_isolation ON post_assets;
DROP POLICY IF EXISTS assets_isolation ON assets;

ALTER TABLE post_autosaves DISABLE ROW LEVEL SECURITY;
ALTER TABLE post_assets DISABLE ROW LEVEL SECURITY;
ALTER TABLE assets DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_post_autosaves_user;
DROP INDEX IF EXISTS idx_post_autosaves_post;
DROP TABLE IF EXISTS post_autosaves CASCADE;

DROP INDEX IF EXISTS idx_post_assets_asset;
DROP INDEX IF EXISTS idx_post_assets_post;
DROP TABLE IF EXISTS post_assets CASCADE;

DROP INDEX IF EXISTS idx_assets_search;
DROP INDEX IF EXISTS idx_assets_created;
DROP INDEX IF EXISTS idx_assets_extension;
DROP INDEX IF EXISTS idx_assets_mime;
DROP INDEX IF EXISTS idx_assets_user;
DROP INDEX IF EXISTS idx_assets_site;
DROP TABLE IF EXISTS assets CASCADE;
