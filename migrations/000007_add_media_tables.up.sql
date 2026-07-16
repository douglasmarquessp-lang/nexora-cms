-- 000007_add_media_tables.up.sql
-- Media Library, Variants, Folders

CREATE TABLE folders (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id       UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    parent_id     UUID REFERENCES folders(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL,
    description   TEXT DEFAULT '',
    sort_order    INT NOT NULL DEFAULT 0,
    created_by    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    UNIQUE(site_id, slug)
);

CREATE INDEX idx_folders_site ON folders(site_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_parent ON folders(parent_id) WHERE deleted_at IS NULL;

CREATE TABLE media (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id          UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    folder_id        UUID REFERENCES folders(id) ON DELETE SET NULL,
    filename         VARCHAR(500) NOT NULL,
    original_name    VARCHAR(500) NOT NULL,
    mime_type        VARCHAR(127) NOT NULL,
    extension        VARCHAR(20) NOT NULL,
    size             BIGINT NOT NULL DEFAULT 0,
    width            INT,
    height           INT,
    duration         INT DEFAULT 0,
    hash             VARCHAR(64) NOT NULL DEFAULT '',
    alt_text         VARCHAR(500) DEFAULT '',
    caption          TEXT DEFAULT '',
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'local',
    storage_key      VARCHAR(1000) NOT NULL,
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_by       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deleted_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_site ON media(site_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_media_folder ON media(folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_media_mime ON media(site_id, mime_type);
CREATE INDEX idx_media_extension ON media(site_id, extension);
CREATE INDEX idx_media_hash ON media(hash);
CREATE INDEX idx_media_created ON media(site_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_media_search ON media USING gin(to_tsvector('portuguese', original_name || ' ' || COALESCE(alt_text, '') || ' ' || COALESCE(caption, '')));

CREATE TYPE media_variant_type AS ENUM ('thumbnail', 'small', 'medium', 'large', 'original');

CREATE TABLE media_variants (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id    UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    variant     media_variant_type NOT NULL,
    width       INT NOT NULL DEFAULT 0,
    height      INT NOT NULL DEFAULT 0,
    file_size   BIGINT NOT NULL DEFAULT 0,
    mime_type   VARCHAR(127) NOT NULL,
    storage_key VARCHAR(1000) NOT NULL,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(media_id, variant, mime_type)
);

CREATE INDEX idx_media_variants_media ON media_variants(media_id);

ALTER TABLE media ENABLE ROW LEVEL SECURITY;
ALTER TABLE folders ENABLE ROW LEVEL SECURITY;
ALTER TABLE media_variants ENABLE ROW LEVEL SECURITY;

CREATE POLICY media_isolation ON media
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY folders_isolation ON folders
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY media_variants_isolation ON media_variants
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM media m WHERE m.id = media_id
            AND m.site_id = current_setting('app.current_site_id')::UUID
        )
    );

CREATE TRIGGER set_media_updated_at
    BEFORE UPDATE ON media
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_folders_updated_at
    BEFORE UPDATE ON folders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
