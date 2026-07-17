CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS research_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    topic VARCHAR(500) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    category VARCHAR(100) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    sources_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_research_jobs_site_id ON research_jobs(site_id);
CREATE INDEX idx_research_jobs_status ON research_jobs(status);
CREATE INDEX idx_research_jobs_topic ON research_jobs(topic);
CREATE INDEX idx_research_jobs_language ON research_jobs(language);

CREATE TABLE IF NOT EXISTS research_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    research_job_id UUID NOT NULL REFERENCES research_jobs(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    language VARCHAR(10) DEFAULT '',
    author VARCHAR(255) DEFAULT '',
    published_at TIMESTAMPTZ,
    summary TEXT DEFAULT '',
    main_facts TEXT DEFAULT '',
    statistics TEXT DEFAULT '',
    relevance_score INT DEFAULT 0,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_research_sources_job_id ON research_sources(research_job_id);

CREATE TABLE IF NOT EXISTS research_entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    research_job_id UUID NOT NULL REFERENCES research_jobs(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    name VARCHAR(500) NOT NULL,
    context TEXT DEFAULT '',
    source_url TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_research_entities_job_id ON research_entities(research_job_id);
CREATE INDEX idx_research_entities_type ON research_entities(entity_type);

CREATE TABLE IF NOT EXISTS research_briefings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    research_job_id UUID NOT NULL REFERENCES research_jobs(id) ON DELETE CASCADE,
    structured_briefing JSONB DEFAULT '{}'::jsonb,
    timeline JSONB DEFAULT '[]'::jsonb,
    confirmed_facts JSONB DEFAULT '[]'::jsonb,
    conflicting_info JSONB DEFAULT '[]'::jsonb,
    editorial_approaches JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_research_briefings_job_id ON research_briefings(research_job_id);
