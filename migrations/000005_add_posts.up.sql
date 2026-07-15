-- 000005_add_posts.up.sql
-- Articles, Categories, Tags and relations

-- ============ POSTS / ARTICLES ============

CREATE TABLE posts (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id      UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    title        VARCHAR(500) NOT NULL,
    slug         VARCHAR(500) NOT NULL,
    content      JSONB NOT NULL DEFAULT '[]',
    excerpt      TEXT DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'draft'
                   CHECK (status IN ('draft', 'published', 'scheduled', 'archived')),
    author_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ,
    scheduled_at TIMESTAMPTZ,
    post_meta    JSONB NOT NULL DEFAULT '{}',
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    UNIQUE(site_id, slug)
);

CREATE INDEX idx_posts_site_status ON posts(site_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_site_slug ON posts(site_id, slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_published ON posts(site_id, published_at DESC) WHERE status = 'published';
CREATE INDEX idx_posts_created ON posts(site_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_search ON posts USING gin(to_tsvector('portuguese', title || ' ' || COALESCE(excerpt, '')));

-- ============ CATEGORIES ============

CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id     UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES categories(id) ON DELETE SET NULL,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    icon        VARCHAR(100) DEFAULT '',
    color       VARCHAR(20) DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE(site_id, slug)
);

CREATE INDEX idx_categories_site ON categories(site_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_categories_parent ON categories(parent_id);
CREATE INDEX idx_categories_sort ON categories(site_id, sort_order);

-- ============ TAGS ============

CREATE TABLE tags (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL,
    color      VARCHAR(20) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(site_id, slug)
);

CREATE INDEX idx_tags_site ON tags(site_id) WHERE deleted_at IS NULL;

-- ============ RELATIONS ============

CREATE TABLE post_categories (
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, category_id)
);

CREATE INDEX idx_post_categories_category ON post_categories(category_id);

CREATE TABLE post_tags (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

CREATE INDEX idx_post_tags_tag ON post_tags(tag_id);

-- ============ RLS ============

ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE tags ENABLE ROW LEVEL SECURITY;

-- RLS: users see only their site's data
CREATE POLICY posts_isolation ON posts
    FOR ALL
    USING (
        site_id = current_setting('app.current_user_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY categories_isolation ON categories
    FOR ALL
    USING (
        site_id = current_setting('app.current_user_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY tags_isolation ON tags
    FOR ALL
    USING (
        site_id = current_setting('app.current_user_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

-- ============ TRIGGERS ============

CREATE TRIGGER set_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_tags_updated_at
    BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
