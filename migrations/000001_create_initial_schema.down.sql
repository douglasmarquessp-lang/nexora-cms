-- 000001_create_initial_schema.down.sql

DROP TRIGGER IF EXISTS set_users_updated_at ON users;
DROP TRIGGER IF EXISTS set_sites_updated_at ON sites;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS audit_log_default;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS site_users;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS sites;

