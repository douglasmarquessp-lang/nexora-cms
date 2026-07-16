-- 000007_add_media_tables.down.sql

DROP TRIGGER IF EXISTS set_folders_updated_at ON folders;
DROP TRIGGER IF EXISTS set_media_updated_at ON media;

DROP POLICY IF EXISTS media_variants_isolation ON media_variants;
DROP POLICY IF EXISTS folders_isolation ON folders;
DROP POLICY IF EXISTS media_isolation ON media;

ALTER TABLE media_variants DISABLE ROW LEVEL SECURITY;
ALTER TABLE folders DISABLE ROW LEVEL SECURITY;
ALTER TABLE media DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_media_variants_media;
DROP INDEX IF EXISTS idx_media_search;
DROP INDEX IF EXISTS idx_media_created;
DROP INDEX IF EXISTS idx_media_hash;
DROP INDEX IF EXISTS idx_media_extension;
DROP INDEX IF EXISTS idx_media_mime;
DROP INDEX IF EXISTS idx_media_folder;
DROP INDEX IF EXISTS idx_media_site;
DROP INDEX IF EXISTS idx_folders_parent;
DROP INDEX IF EXISTS idx_folders_site;

DROP TABLE IF EXISTS media_variants;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS folders;

DROP TYPE IF EXISTS media_variant_type;
