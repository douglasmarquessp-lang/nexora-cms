-- 000001_create_initial_schema.up.sql
-- Schema inicial do Nexora CMS

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============ SITES ============

CREATE TABLE sites (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    uuid        UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    slug        VARCHAR(255) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    domain      VARCHAR(255),
    settings    JSONB NOT NULL DEFAULT '{}',
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'inactive', 'suspended')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sites_slug ON sites(slug);
CREATE INDEX idx_sites_domain ON sites(domain);

-- ============ USERS ============

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    uuid          UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL DEFAULT '',
    avatar        VARCHAR(500),
    role          VARCHAR(20) NOT NULL DEFAULT 'user'
                  CHECK (role IN ('super_admin', 'user')),
    metadata      JSONB NOT NULL DEFAULT '{}',
    last_login    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

-- ============ SITE USERS (Pivot) ============

CREATE TABLE site_users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id     UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role        VARCHAR(20) NOT NULL DEFAULT 'editor'
                CHECK (role IN ('admin', 'editor', 'author', 'subscriber')),
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, site_id)
);

CREATE INDEX idx_site_users_user ON site_users(user_id);
CREATE INDEX idx_site_users_site ON site_users(site_id);

-- ============ SESSIONS ============

CREATE TABLE sessions (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(500) NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_refresh ON sessions(refresh_token);

-- ============ AUDIT LOG ============

CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID REFERENCES users(id),
    site_id     UUID REFERENCES sites(id),
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id   UUID,
    payload     JSONB NOT NULL DEFAULT '{}',
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_site ON audit_log(site_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at);

-- Default partition for current and future data
CREATE TABLE audit_log_default PARTITION OF audit_log
    FOR VALUES FROM ('2024-01-01') TO ('2030-01-01');

-- ============ TRIGGER: updated_at ============

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_sites_updated_at
    BEFORE UPDATE ON sites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
