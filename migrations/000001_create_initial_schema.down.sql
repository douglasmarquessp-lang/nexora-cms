-- 000001_create_initial_schema.down.sql
-- Rollback initial schema

DROP TRIGGER IF EXISTS set_sites_updated_at ON sites;
DROP TRIGGER IF EXISTS set_users_updated_at ON sites;

DROP TABLE IF EXISTS site_users CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS audit_log_default CASCADE;
DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS sites CASCADE;

-- NOTE: Extensions (uuid-ossp, pg_trgm) are NOT dropped because they are
-- shared dependencies used by subsequent migrations. Dropping them here
-- would break rollback of later migrations in a single-transaction runner.
