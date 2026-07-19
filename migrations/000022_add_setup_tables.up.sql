CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS system_installation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    installed BOOLEAN NOT NULL DEFAULT false,
    installed_at TIMESTAMPTZ,
    cms_name VARCHAR(255) NOT NULL DEFAULT 'Nexora CMS',
    admin_name VARCHAR(255) NOT NULL DEFAULT '',
    admin_email VARCHAR(255) NOT NULL DEFAULT '',
    default_site VARCHAR(255) NOT NULL DEFAULT '',
    version VARCHAR(50) NOT NULL DEFAULT '0.1.0',
    locale VARCHAR(10) NOT NULL DEFAULT 'pt-BR',
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/Sao_Paulo',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_installation_installed ON system_installation(installed);
