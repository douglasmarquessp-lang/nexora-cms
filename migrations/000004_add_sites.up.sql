-- 000004_add_sites.up.sql
-- Multi-site system: sites, domains, settings, feature flags

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============ SITES ============

-- Migration 001 already creates a sites table with a simpler schema.
-- Use IF NOT EXISTS to avoid conflicts, then migrate existing schema forward.
CREATE TABLE IF NOT EXISTS sites (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name         VARCHAR(255) NOT NULL,
    slug         VARCHAR(255) NOT NULL UNIQUE,
    description  TEXT DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active', 'inactive', 'suspended', 'maintenance')),
    owner_id     UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    settings     JSONB NOT NULL DEFAULT '{}',
    feature_flags JSONB NOT NULL DEFAULT '{}',
    theme        VARCHAR(100) DEFAULT 'default',
    locale       VARCHAR(10) DEFAULT 'pt-BR',
    timezone     VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

-- Add columns that migration 001's sites table is missing
ALTER TABLE sites ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS feature_flags JSONB NOT NULL DEFAULT '{}';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS theme VARCHAR(100) DEFAULT 'default';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS locale VARCHAR(10) DEFAULT 'pt-BR';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- owner_id: add nullable first, then set NOT NULL (table is empty at migration time)
ALTER TABLE sites ADD COLUMN IF NOT EXISTS owner_id UUID;
ALTER TABLE sites ALTER COLUMN owner_id SET NOT NULL;
ALTER TABLE sites ADD CONSTRAINT fk_sites_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_sites_slug ON sites(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sites_owner ON sites(owner_id);
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites(status);

-- ============ SITE DOMAINS ============

CREATE TABLE site_domains (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    domain     VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    verified   BOOLEAN NOT NULL DEFAULT false,
    ssl_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(domain)
);

CREATE INDEX idx_site_domains_site ON site_domains(site_id);
CREATE INDEX idx_site_domains_domain ON site_domains(domain);

-- ============ GLOBAL SETTINGS ============

CREATE TABLE global_settings (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key        VARCHAR(255) NOT NULL UNIQUE,
    value      JSONB NOT NULL DEFAULT '{}',
    type       VARCHAR(50) NOT NULL DEFAULT 'string'
                 CHECK (type IN ('string', 'number', 'boolean', 'json', 'array')),
    description TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_global_settings_key ON global_settings(key);

-- ============ SITE SETTINGS (override layer) ============

CREATE TABLE site_settings (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    key        VARCHAR(255) NOT NULL,
    value      JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(site_id, key)
);

CREATE INDEX idx_site_settings_site ON site_settings(site_id, key);

-- ============ CASBIN POLICIES ============

CREATE TABLE casbin_rules (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ptype      VARCHAR(10) NOT NULL,
    v0         VARCHAR(255) NOT NULL DEFAULT '',
    v1         VARCHAR(255) NOT NULL DEFAULT '',
    v2         VARCHAR(255) NOT NULL DEFAULT '',
    v3         VARCHAR(255) NOT NULL DEFAULT '',
    v4         VARCHAR(255) NOT NULL DEFAULT '',
    v5         VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_casbin_rules_ptype ON casbin_rules(ptype);
CREATE INDEX idx_casbin_rules_v0 ON casbin_rules(v0);
CREATE INDEX idx_casbin_rules_v1 ON casbin_rules(v1);

-- ============ DEFAULT FEATURE FLAGS ============

INSERT INTO global_settings (key, value, type, description) VALUES
    ('site.max_sites_per_user', '5', 'number', 'Maximum number of sites a user can create'),
    ('site.default_locale', '"pt-BR"', 'string', 'Default locale for new sites'),
    ('site.default_timezone', '"America/Sao_Paulo"', 'string', 'Default timezone for new sites'),
    ('auth.registration_enabled', 'true', 'boolean', 'Allow new user registrations'),
    ('auth.mfa_required', 'false', 'boolean', 'Require MFA for all users'),
    ('features.seo_module', 'true', 'boolean', 'Enable SEO module'),
    ('features.ai_module', 'false', 'boolean', 'Enable AI module'),
    ('features.analytics', 'true', 'boolean', 'Enable analytics'),
    ('features.api_public', 'false', 'boolean', 'Enable public API'),
    ('storage.max_upload_size_mb', '50', 'number', 'Maximum upload file size in MB');

-- ============ RLS ============

ALTER TABLE sites ENABLE ROW LEVEL SECURITY;
ALTER TABLE site_domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE site_settings ENABLE ROW LEVEL SECURITY;

-- RLS policies: users can only see their own sites
-- SuperAdmins (role = 'superadmin') can see all
CREATE POLICY sites_isolation ON sites
    FOR ALL
    USING (
        owner_id = current_setting('app.current_user_id')::UUID
        OR current_setting('app.current_user_role') = 'superadmin'
        OR current_setting('app.current_user_role') = 'siteadmin'
    );

CREATE POLICY site_domains_isolation ON site_domains
    FOR ALL
    USING (
        site_id IN (
            SELECT id FROM sites WHERE owner_id = current_setting('app.current_user_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY site_settings_isolation ON site_settings
    FOR ALL
    USING (
        site_id IN (
            SELECT id FROM sites WHERE owner_id = current_setting('app.current_user_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

-- ============ TRIGGERS ============

DROP TRIGGER IF EXISTS set_sites_updated_at ON sites;
CREATE TRIGGER set_sites_updated_at
    BEFORE UPDATE ON sites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_site_domains_updated_at
    BEFORE UPDATE ON site_domains
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_global_settings_updated_at
    BEFORE UPDATE ON global_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_site_settings_updated_at
    BEFORE UPDATE ON site_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
