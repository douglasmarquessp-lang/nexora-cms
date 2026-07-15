-- 000002_add_oauth_mfa.down.sql

DROP TRIGGER IF EXISTS set_mfa_configs_updated_at ON mfa_configs;
DROP TRIGGER IF EXISTS set_oauth_accounts_updated_at ON oauth_accounts;

DROP INDEX IF EXISTS idx_mfa_configs_user;
DROP INDEX IF EXISTS idx_oauth_accounts_provider;
DROP INDEX IF EXISTS idx_oauth_accounts_user;

DROP TABLE IF EXISTS mfa_configs;
DROP TABLE IF EXISTS oauth_accounts;

ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_secret;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled;
