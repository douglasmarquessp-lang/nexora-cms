CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS publications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    post_id UUID,
    title VARCHAR(500) NOT NULL,
    content TEXT DEFAULT '',
    excerpt TEXT DEFAULT '',
    slug VARCHAR(500) NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    canonical_url TEXT DEFAULT '',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    translations JSONB DEFAULT '{}'::jsonb,
    multilingual_urls JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    visibility VARCHAR(20) NOT NULL DEFAULT 'public',
    author_id UUID,
    published_by UUID,
    published_at TIMESTAMPTZ,
    unpublished_at TIMESTAMPTZ,
    scheduled_at TIMESTAMPTZ,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    meta_title VARCHAR(500) DEFAULT '',
    meta_description TEXT DEFAULT '',
    og_image TEXT DEFAULT '',
    featured_image_url TEXT DEFAULT '',
    tags TEXT[] DEFAULT '{}',
    categories TEXT[] DEFAULT '{}',
    word_count INT DEFAULT 0,
    reading_time INT DEFAULT 0,
    revision INT NOT NULL DEFAULT 1,
    checksum VARCHAR(64) DEFAULT '',
    source VARCHAR(100) DEFAULT 'manual',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS publication_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    publication_id UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    previous_status VARCHAR(50) DEFAULT '',
    new_status VARCHAR(50) DEFAULT '',
    title VARCHAR(500) DEFAULT '',
    slug VARCHAR(500) DEFAULT '',
    changes JSONB DEFAULT '{}'::jsonb,
    reason TEXT DEFAULT '',
    performed_by UUID,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS publication_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    publication_id UUID REFERENCES publications(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL DEFAULT 'publish',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 5,
    scheduled_for TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT DEFAULT '',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS publication_schedule (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    publication_id UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL,
    action VARCHAR(50) NOT NULL DEFAULT 'publish',
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    recurrence VARCHAR(50) DEFAULT '',
    recurrence_end TIMESTAMPTZ,
    notify_on_publish BOOLEAN NOT NULL DEFAULT false,
    notify_users UUID[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by UUID,
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS publication_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    publication_id UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    view_count BIGINT NOT NULL DEFAULT 0,
    unique_visitors BIGINT NOT NULL DEFAULT 0,
    avg_time_seconds NUMERIC(10,2) DEFAULT 0,
    bounce_rate NUMERIC(5,2) DEFAULT 0,
    share_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    like_count INT NOT NULL DEFAULT 0,
    click_count INT NOT NULL DEFAULT 0,
    ctr NUMERIC(5,2) DEFAULT 0,
    scroll_depth NUMERIC(5,2) DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pub_site ON publications(site_id);
CREATE INDEX idx_pub_slug ON publications(site_id, slug);
CREATE INDEX idx_pub_status ON publications(status);
CREATE INDEX idx_pub_language ON publications(language);
CREATE INDEX idx_pub_published_at ON publications(published_at);
CREATE INDEX idx_pub_scheduled_at ON publications(scheduled_at);
CREATE INDEX idx_pub_author ON publications(author_id);
CREATE INDEX idx_pub_history_publication ON publication_history(publication_id);
CREATE INDEX idx_pub_history_site ON publication_history(site_id);
CREATE INDEX idx_pub_history_action ON publication_history(action);
CREATE INDEX idx_pub_queue_site ON publication_queue(site_id);
CREATE INDEX idx_pub_queue_status ON publication_queue(status);
CREATE INDEX idx_pub_queue_scheduled ON publication_queue(scheduled_for);
CREATE INDEX idx_pub_schedule_site ON publication_schedule(site_id);
CREATE INDEX idx_pub_schedule_publication ON publication_schedule(publication_id);
CREATE INDEX idx_pub_schedule_at ON publication_schedule(scheduled_at);
CREATE INDEX idx_pub_schedule_status ON publication_schedule(status);
CREATE INDEX idx_pub_metrics_publication ON publication_metrics(publication_id);
CREATE INDEX idx_pub_metrics_site ON publication_metrics(site_id);
CREATE UNIQUE INDEX idx_pub_unique_slug ON publications(site_id, slug) WHERE status != 'deleted';
