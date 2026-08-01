-- Fix: recreate Casbin rules table if it is missing
CREATE TABLE IF NOT EXISTS casbin_rules (
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

CREATE INDEX IF NOT EXISTS idx_casbin_rules_ptype
    ON casbin_rules(ptype);

CREATE INDEX IF NOT EXISTS idx_casbin_rules_v0
    ON casbin_rules(v0);

CREATE INDEX IF NOT EXISTS idx_casbin_rules_v1
    ON casbin_rules(v1);
