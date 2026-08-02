-- 000003_add_audit_log.up.sql
--
-- NOTE: migration 000001 already creates audit_log (partitioned) plus the
-- indexes idx_audit_log_user and idx_audit_log_created. This migration only
-- fills the gap (idx_audit_log_action, idx_audit_log_entity), so every index
-- must be IF NOT EXISTS to avoid duplicate-index failures on fresh installs.

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    site_id     UUID,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100),
    entity_id   UUID,
    payload     JSONB DEFAULT '{}',
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id);
