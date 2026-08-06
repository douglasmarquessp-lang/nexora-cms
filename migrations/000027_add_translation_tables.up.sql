-- 000027_add_translation_tables.up.sql
-- Translation Intelligence Engine (Sprint 5.7)
--
-- Tables:
--   translation_jobs   — one translation run (source article/context → target language/site)
--   translation_stages — pipeline stages (translate, native_review, seo_review, publish)
--   glossary_terms     — persistent glossary (global when project_id IS NULL, per-project otherwise)
--
-- Design notes:
--   - A job lives on the SOURCE site (site_id) and produces a post on the
--     TARGET site (target_site_id). Both are always explicit in every query
--     (RLS isolation is defense-in-depth; the pipeline goroutine runs outside
--     the request's RLS context, so all SQL carries explicit site filters).
--   - Each stage can reject the previous stage: a rejected stage moves the job
--     back to the previous stage with feedback (see translation_stages.status
--     = 'rejected' and translation_jobs.current_stage).
--   - glossary_terms.project_id NULL = global glossary term (all projects on
--     the site); otherwise scoped to one project.

-- ============================================================
-- 1. TRANSLATION JOBS
-- ============================================================

CREATE TABLE translation_jobs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id           UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    project_id        UUID,
    source_post_id    UUID,
    target_site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    source_language   VARCHAR(8) NOT NULL DEFAULT 'pt',
    target_language   VARCHAR(8) NOT NULL DEFAULT 'en',
    title             TEXT NOT NULL,
    content           TEXT NOT NULL DEFAULT '',
    status            VARCHAR(32) NOT NULL DEFAULT 'pending',
    current_stage     VARCHAR(32),
    translation_score JSONB DEFAULT NULL,
    published_post_id UUID,
    publication_id    UUID,
    error_message     TEXT,
    created_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ
);

CREATE INDEX idx_translation_jobs_site ON translation_jobs (site_id, created_at DESC);
CREATE INDEX idx_translation_jobs_target_site ON translation_jobs (target_site_id);
CREATE INDEX idx_translation_jobs_status ON translation_jobs (status);

CREATE TRIGGER set_updated_at_translation_jobs
    BEFORE UPDATE ON translation_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 2. TRANSLATION STAGES
-- ============================================================

CREATE TABLE translation_stages (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    translation_job_id UUID NOT NULL REFERENCES translation_jobs(id) ON DELETE CASCADE,
    stage             VARCHAR(32) NOT NULL,
    status            VARCHAR(32) NOT NULL DEFAULT 'pending',
    score             NUMERIC(5,2),
    attempt           INTEGER NOT NULL DEFAULT 1,
    feedback          TEXT,
    result            JSONB DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ
);

CREATE INDEX idx_translation_stages_job ON translation_stages (translation_job_id, stage, attempt);

CREATE TRIGGER set_updated_at_translation_stages
    BEFORE UPDATE ON translation_stages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 3. GLOSSARY TERMS
-- ============================================================

CREATE TABLE glossary_terms (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id          UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    project_id       UUID,
    source_term      TEXT NOT NULL,
    target_term      TEXT NOT NULL,
    source_language  VARCHAR(8) NOT NULL DEFAULT 'pt',
    target_language  VARCHAR(8) NOT NULL DEFAULT 'en',
    forbidden        BOOLEAN NOT NULL DEFAULT FALSE,
    description      TEXT,
    created_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_glossary_terms_site ON glossary_terms (site_id, project_id);
CREATE UNIQUE INDEX idx_glossary_terms_unique ON glossary_terms
    (site_id, project_id, LOWER(source_term), source_language, target_language)
    WHERE forbidden = FALSE;
CREATE UNIQUE INDEX idx_glossary_terms_forbidden_unique ON glossary_terms
    (site_id, project_id, LOWER(source_term), source_language, target_language)
    WHERE forbidden = TRUE;

CREATE TRIGGER set_updated_at_glossary_terms
    BEFORE UPDATE ON glossary_terms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 4. ROW LEVEL SECURITY
-- ============================================================

ALTER TABLE translation_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE translation_stages ENABLE ROW LEVEL SECURITY;
ALTER TABLE glossary_terms ENABLE ROW LEVEL SECURITY;

CREATE POLICY translation_jobs_isolation ON translation_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY translation_stages_isolation ON translation_stages
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM translation_jobs tj WHERE tj.id = translation_job_id
            AND tj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY glossary_terms_isolation ON glossary_terms
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );
