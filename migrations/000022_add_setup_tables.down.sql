-- 000022_add_setup_tables.down.sql
-- Rollback system installation setup table

DROP INDEX IF EXISTS idx_system_installation_installed;
DROP TABLE IF EXISTS system_installation CASCADE;
