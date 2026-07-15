-- 000004_add_sites.down.sql

DROP TRIGGER IF EXISTS set_site_settings_updated_at ON site_settings;
DROP TRIGGER IF EXISTS set_global_settings_updated_at ON global_settings;
DROP TRIGGER IF EXISTS set_site_domains_updated_at ON site_domains;
DROP TRIGGER IF EXISTS set_sites_updated_at ON sites;

DROP POLICY IF EXISTS site_settings_isolation ON site_settings;
DROP POLICY IF EXISTS site_domains_isolation ON site_domains;
DROP POLICY IF EXISTS sites_isolation ON sites;

ALTER TABLE site_settings DISABLE ROW LEVEL SECURITY;
ALTER TABLE site_domains DISABLE ROW LEVEL SECURITY;
ALTER TABLE sites DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_casbin_rules_v1;
DROP INDEX IF EXISTS idx_casbin_rules_v0;
DROP INDEX IF EXISTS idx_casbin_rules_ptype;
DROP INDEX IF EXISTS idx_site_settings_site;
DROP INDEX IF EXISTS idx_global_settings_key;
DROP INDEX IF EXISTS idx_site_domains_domain;
DROP INDEX IF EXISTS idx_site_domains_site;
DROP INDEX IF EXISTS idx_sites_status;
DROP INDEX IF EXISTS idx_sites_owner;
DROP INDEX IF EXISTS idx_sites_slug;

DROP TABLE IF EXISTS casbin_rules;
DROP TABLE IF EXISTS site_settings;
DROP TABLE IF EXISTS global_settings;
DROP TABLE IF EXISTS site_domains;
DROP TABLE IF EXISTS sites;
