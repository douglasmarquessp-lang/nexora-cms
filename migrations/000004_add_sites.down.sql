-- 000004_add_sites.down.sql
-- Rollback multi-site system additions.
--
-- NOTE: The sites table was created in 000001. This migration only rolled
-- forward by adding columns and creating companion tables. The down
-- reverses ONLY what 000004 did, leaving the core sites table intact
-- for 000001.down to handle.

-- RLS policies
DROP POLICY IF EXISTS site_settings_isolation ON site_settings;
DROP POLICY IF EXISTS site_domains_isolation ON site_domains;
DROP POLICY IF EXISTS sites_isolation ON sites;

-- Disable RLS (only if no other migration re-enabled it; safe to call twice)
ALTER TABLE site_settings DISABLE ROW LEVEL SECURITY;
ALTER TABLE site_domains DISABLE ROW LEVEL SECURITY;
ALTER TABLE sites DISABLE ROW LEVEL SECURITY;

-- Seed data
DELETE FROM global_settings WHERE key IN (
    'site.max_sites_per_user',
    'site.default_locale',
    'site.default_timezone',
    'auth.registration_enabled',
    'auth.mfa_required',
    'features.seo_module',
    'features.ai_module',
    'features.analytics',
    'features.api_public',
    'storage.max_upload_size_mb'
);

-- Casbin tables
DROP INDEX IF EXISTS idx_casbin_rules_v1;
DROP INDEX IF EXISTS idx_casbin_rules_v0;
DROP INDEX IF EXISTS idx_casbin_rules_ptype;
DROP TABLE IF EXISTS casbin_rules CASCADE;

-- Site settings
DROP INDEX IF EXISTS idx_site_settings_site;
DROP TABLE IF EXISTS site_settings CASCADE;

-- Global settings
DROP INDEX IF EXISTS idx_global_settings_key;
DROP TABLE IF EXISTS global_settings CASCADE;

-- Site domains
DROP INDEX IF EXISTS idx_site_domains_domain;
DROP INDEX IF EXISTS idx_site_domains_site;
DROP TABLE IF EXISTS site_domains CASCADE;

-- Columns added to sites table
ALTER TABLE sites DROP CONSTRAINT IF EXISTS fk_sites_owner;
ALTER TABLE sites ALTER COLUMN owner_id DROP NOT NULL;
ALTER TABLE sites DROP COLUMN IF EXISTS owner_id;
ALTER TABLE sites DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE sites DROP COLUMN IF EXISTS timezone;
ALTER TABLE sites DROP COLUMN IF EXISTS locale;
ALTER TABLE sites DROP COLUMN IF EXISTS theme;
ALTER TABLE sites DROP COLUMN IF EXISTS feature_flags;
ALTER TABLE sites DROP COLUMN IF EXISTS description;

-- Indexes added by this migration
DROP INDEX IF EXISTS idx_sites_status;
DROP INDEX IF EXISTS idx_sites_owner;
DROP INDEX IF EXISTS idx_sites_slug;

-- 000004 DROP'd and re-CREATEd this trigger; restore original from 000001
DROP TRIGGER IF EXISTS set_sites_updated_at ON sites;
CREATE TRIGGER set_sites_updated_at
    BEFORE UPDATE ON sites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Triggers created by this migration
DROP TRIGGER IF EXISTS set_site_settings_updated_at ON site_settings;
DROP TRIGGER IF EXISTS set_global_settings_updated_at ON global_settings;
DROP TRIGGER IF EXISTS set_site_domains_updated_at ON site_domains;
