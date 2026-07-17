CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS editorial_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date TIMESTAMPTZ,
    post_id UUID REFERENCES posts(id) ON DELETE SET NULL,
    created_by UUID,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_editorial_tasks_site_id ON editorial_tasks(site_id);
CREATE INDEX idx_editorial_tasks_status ON editorial_tasks(status);
CREATE INDEX idx_editorial_tasks_due_date ON editorial_tasks(due_date);

CREATE TABLE IF NOT EXISTS post_revisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    version INT NOT NULL,
    title TEXT NOT NULL,
    content JSONB DEFAULT '[]'::jsonb,
    excerpt TEXT DEFAULT '',
    slug TEXT NOT NULL,
    post_meta JSONB DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    summary TEXT,
    change_log TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_post_revisions_post_id ON post_revisions(post_id);
CREATE INDEX idx_post_revisions_site_id ON post_revisions(site_id);
CREATE UNIQUE INDEX idx_post_revisions_version ON post_revisions(post_id, version);

CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    comments TEXT,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approval_requests_post_id ON approval_requests(post_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);

CREATE TABLE IF NOT EXISTS editorial_calendar_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    event_date DATE NOT NULL,
    event_type VARCHAR(50) NOT NULL DEFAULT 'publication',
    post_id UUID REFERENCES posts(id) ON DELETE SET NULL,
    color VARCHAR(20),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendar_events_site_date ON editorial_calendar_events(site_id, event_date);

CREATE TABLE IF NOT EXISTS editorial_widgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    widget_type VARCHAR(100) NOT NULL,
    title VARCHAR(200) NOT NULL,
    config JSONB DEFAULT '{}'::jsonb,
    position INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_editorial_widgets_site_id ON editorial_widgets(site_id);

INSERT INTO editorial_widgets (site_id, widget_type, title, config, position, enabled)
SELECT s.id, 'stats', 'Statistics', '{}'::jsonb, 0, true FROM sites s;

INSERT INTO editorial_widgets (site_id, widget_type, title, config, position, enabled)
SELECT s.id, 'recent_posts', 'Recent Articles', '{"limit":5}'::jsonb, 1, true FROM sites s;

INSERT INTO editorial_widgets (site_id, widget_type, title, config, position, enabled)
SELECT s.id, 'draft_posts', 'Drafts', '{"limit":5}'::jsonb, 2, true FROM sites s;

INSERT INTO editorial_widgets (site_id, widget_type, title, config, position, enabled)
SELECT s.id, 'scheduled_posts', 'Scheduled', '{"limit":5}'::jsonb, 3, true FROM sites s;

INSERT INTO editorial_widgets (site_id, widget_type, title, config, position, enabled)
SELECT s.id, 'tasks', 'Tasks', '{"limit":5}'::jsonb, 4, true FROM sites s;

INSERT INTO editorial_widgets (site_id, widget_type, title, config, position, enabled)
SELECT s.id, 'calendar', 'Editorial Calendar', '{}'::jsonb, 5, true FROM sites s;
