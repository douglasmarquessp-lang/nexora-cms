-- 000008_add_plugins.down.sql
-- Rollback plugin system tables

DROP TRIGGER IF EXISTS set_plugin_settings_updated_at ON plugin_settings;
DROP TRIGGER IF EXISTS set_plugins_updated_at ON plugins;

DROP INDEX IF EXISTS idx_plugins_plugin_id;
DROP INDEX IF EXISTS idx_plugins_status;

DROP TABLE IF EXISTS plugin_permissions CASCADE;
DROP TABLE IF EXISTS plugin_settings CASCADE;
DROP TABLE IF EXISTS plugins CASCADE;
