-- 000002_add_oauth_mfa.up.sql
-- OAuth accounts and MFA support

-- ============ OAUTH ACCOUNTS ============

CREATE TABLE oauth_accounts (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      VARCHAR(50) NOT NULL,
    provider_id   VARCHAR(255) NOT NULL,
    access_token  TEXT,
    refresh_token TEXT,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

CREATE INDEX idx_oauth_accounts_user ON oauth_accounts(user_id);
CREATE INDEX idx_oauth_accounts_provider ON oauth_accounts(provider, provider_id);

-- ============ MFA CONFIGURATIONS ============

CREATE TABLE mfa_configs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    secret        VARCHAR(255) NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT false,
    method        VARCHAR(20) NOT NULL DEFAULT 'totp',
    backup_codes  TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mfa_configs_user ON mfa_configs(user_id);

-- ============ ALTER USERS TABLE ============

ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX idx_users_deleted ON users(deleted_at) WHERE deleted_at IS NULL;

-- ============ TRIGGERS ============

CREATE TRIGGER set_oauth_accounts_updated_at
    BEFORE UPDATE ON oauth_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_mfa_configs_updated_at
    BEFORE UPDATE ON mfa_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
