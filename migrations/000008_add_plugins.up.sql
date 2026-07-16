-- 000008_add_plugins.up.sql
-- Plugin System tables

CREATE TABLE plugins (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plugin_id       VARCHAR(255) NOT NULL UNIQUE,
    name            VARCHAR(255) NOT NULL,
    version         VARCHAR(50) NOT NULL,
    author          VARCHAR(255) NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    license         VARCHAR(100) NOT NULL DEFAULT '',
    homepage        VARCHAR(500) NOT NULL DEFAULT '',
    min_core_version VARCHAR(50) NOT NULL DEFAULT '',
    dependencies    JSONB NOT NULL DEFAULT '[]',
    permissions     JSONB NOT NULL DEFAULT '[]',
    hooks           JSONB NOT NULL DEFAULT '[]',
    admin_pages     JSONB NOT NULL DEFAULT '[]',
    status          VARCHAR(20) NOT NULL DEFAULT 'installed' CHECK (status IN ('installed', 'active', 'inactive')),
    installed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_plugins_status ON plugins(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_plugins_plugin_id ON plugins(plugin_id) WHERE deleted_at IS NULL;

CREATE TABLE plugin_settings (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plugin_id   UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    value       JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, key)
);

CREATE TABLE plugin_permissions (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plugin_id     UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    permission    VARCHAR(255) NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    default_roles JSONB NOT NULL DEFAULT '[]',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, permission)
);

CREATE TRIGGER set_plugins_updated_at
    BEFORE UPDATE ON plugins
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_plugin_settings_updated_at
    BEFORE UPDATE ON plugin_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
