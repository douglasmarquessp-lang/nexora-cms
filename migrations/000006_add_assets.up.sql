-- 000006_add_assets.up.sql
-- Assets, Post_Assets and storage tracking

-- ============ ASSETS ============

CREATE TABLE assets (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id          UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename         VARCHAR(500) NOT NULL,
    original_name    VARCHAR(500) NOT NULL,
    mime_type        VARCHAR(127) NOT NULL,
    extension        VARCHAR(20) NOT NULL,
    size             BIGINT NOT NULL DEFAULT 0,
    width            INT,
    height           INT,
    alt_text         VARCHAR(500) DEFAULT '',
    title            VARCHAR(500) DEFAULT '',
    caption          TEXT DEFAULT '',
    description      TEXT DEFAULT '',
    thumbnail_path   VARCHAR(1000) DEFAULT '',
    optimized_path   VARCHAR(1000) DEFAULT '',
    storage_provider VARCHAR(20) NOT NULL DEFAULT 'local',
    storage_path     VARCHAR(1000) NOT NULL,
    url              VARCHAR(2000) DEFAULT '',
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_assets_site ON assets(site_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_user ON assets(user_id);
CREATE INDEX idx_assets_mime ON assets(site_id, mime_type);
CREATE INDEX idx_assets_extension ON assets(site_id, extension);
CREATE INDEX idx_assets_created ON assets(site_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_search ON assets USING gin(to_tsvector('portuguese', original_name || ' ' || COALESCE(alt_text, '') || ' ' || COALESCE(title, '') || ' ' || COALESCE(caption, '')));

-- ============ POST ASSETS (relacionamento) ============

CREATE TABLE post_assets (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    asset_id   UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0,
    type       VARCHAR(30) NOT NULL DEFAULT 'gallery'
                 CHECK (type IN ('featured_image', 'gallery', 'attachment')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(post_id, asset_id)
);

CREATE INDEX idx_post_assets_post ON post_assets(post_id);
CREATE INDEX idx_post_assets_asset ON post_assets(asset_id);

-- ============ AUTOSAVE ============

CREATE TABLE post_autosaves (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    JSONB NOT NULL DEFAULT '[]',
    title      VARCHAR(500) DEFAULT '',
    excerpt    TEXT DEFAULT '',
    metadata   JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(post_id, user_id)
);

CREATE INDEX idx_post_autosaves_post ON post_autosaves(post_id);
CREATE INDEX idx_post_autosaves_user ON post_autosaves(user_id);

-- ============ RLS ============

ALTER TABLE assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE post_assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE post_autosaves ENABLE ROW LEVEL SECURITY;

CREATE POLICY assets_isolation ON assets
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY post_assets_isolation ON post_assets
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM posts p WHERE p.id = post_id
            AND p.site_id = current_setting('app.current_site_id')::UUID
        )
    );

CREATE POLICY post_autosaves_isolation ON post_autosaves
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        AND user_id = current_setting('app.current_user_id')::UUID
    );

-- ============ TRIGGERS ============

CREATE TRIGGER set_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
