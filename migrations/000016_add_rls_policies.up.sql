-- 000016_add_rls_policies.up.sql
-- Fix buggy RLS policies and add missing RLS for all tables

-- ============================================================
-- 1. FIX BUGGY RLS POLICIES (migration 000005 used current_user_id
--    instead of current_site_id for site_id comparison)
-- ============================================================

DROP POLICY IF EXISTS posts_isolation ON posts;
DROP POLICY IF EXISTS categories_isolation ON categories;
DROP POLICY IF EXISTS tags_isolation ON tags;

CREATE POLICY posts_isolation ON posts
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY categories_isolation ON categories
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY tags_isolation ON tags
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

-- ============================================================
-- 2. ADD RLS FOR EDITORIAL TABLES (migration 000009)
-- ============================================================

ALTER TABLE editorial_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE post_revisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE approval_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_calendar_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_widgets ENABLE ROW LEVEL SECURITY;

CREATE POLICY editorial_tasks_isolation ON editorial_tasks
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY post_revisions_isolation ON post_revisions
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY approval_requests_isolation ON approval_requests
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY editorial_calendar_events_isolation ON editorial_calendar_events
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY editorial_widgets_isolation ON editorial_widgets
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

-- ============================================================
-- 3. ADD RLS FOR RESEARCH TABLES (migration 000010)
-- ============================================================

ALTER TABLE research_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE research_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE research_entities ENABLE ROW LEVEL SECURITY;
ALTER TABLE research_briefings ENABLE ROW LEVEL SECURITY;

CREATE POLICY research_jobs_isolation ON research_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY research_sources_isolation ON research_sources
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM research_jobs rj WHERE rj.id = job_id
            AND rj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY research_entities_isolation ON research_entities
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM research_jobs rj WHERE rj.id = job_id
            AND rj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY research_briefings_isolation ON research_briefings
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM research_jobs rj WHERE rj.id = job_id
            AND rj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

-- ============================================================
-- 4. ADD RLS FOR WRITER TABLES (migration 000011)
-- ============================================================

ALTER TABLE writing_styles ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_outlines ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_sections ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_versions ENABLE ROW LEVEL SECURITY;

CREATE POLICY writing_styles_isolation ON writing_styles
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY article_jobs_isolation ON article_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY article_outlines_isolation ON article_outlines
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY article_sections_isolation ON article_sections
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY article_versions_isolation ON article_versions
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

-- ============================================================
-- 5. ADD RLS FOR EDITORIAL ENGINE TABLES (migration 000012)
-- ============================================================

ALTER TABLE editorial_pipelines ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_style_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_seo_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_quality_scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_translations ENABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_prompt_data ENABLE ROW LEVEL SECURITY;

CREATE POLICY editorial_pipelines_isolation ON editorial_pipelines
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY pipeline_stages_isolation ON pipeline_stages
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM editorial_pipelines ep WHERE ep.id = pipeline_id
            AND ep.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY editorial_style_rules_isolation ON editorial_style_rules
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY editorial_seo_data_isolation ON editorial_seo_data
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY editorial_quality_scores_isolation ON editorial_quality_scores
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY editorial_translations_isolation ON editorial_translations
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY editorial_prompt_data_isolation ON editorial_prompt_data
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

-- ============================================================
-- 6. ADD RLS FOR GENERATION TABLES (migration 000013)
-- ============================================================

ALTER TABLE generation_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_pipeline ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_pipeline_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_quality_gates ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_stats ENABLE ROW LEVEL SECURITY;

CREATE POLICY generation_jobs_isolation ON generation_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY generation_pipeline_isolation ON generation_pipeline
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM generation_jobs gj WHERE gj.id = job_id
            AND gj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY generation_pipeline_logs_isolation ON generation_pipeline_logs
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM generation_pipeline gp WHERE gp.id = pipeline_id
        )
        AND EXISTS (
            SELECT 1 FROM generation_jobs gj
            INNER JOIN generation_pipeline gp2 ON gp2.job_id = gj.id
            WHERE gp2.id = pipeline_id
            AND gj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY generation_quality_gates_isolation ON generation_quality_gates
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM generation_jobs gj WHERE gj.id = job_id
            AND gj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY generation_stats_isolation ON generation_stats
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

-- ============================================================
-- 7. ADD RLS FOR AUTOCONTENT TABLES (migration 000014)
-- ============================================================

ALTER TABLE autocontent_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE autocontent_steps ENABLE ROW LEVEL SECURITY;
ALTER TABLE autocontent_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE publication_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_templates ENABLE ROW LEVEL SECURITY;

CREATE POLICY autocontent_jobs_isolation ON autocontent_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY autocontent_steps_isolation ON autocontent_steps
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM autocontent_jobs aj WHERE aj.id = job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY autocontent_results_isolation ON autocontent_results
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM autocontent_jobs aj2
            INNER JOIN autocontent_steps aas2 ON aas2.job_id = aj2.id
            WHERE aas2.id = step_id
            AND aj2.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY publication_queue_isolation ON publication_queue
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY workflow_templates_isolation ON workflow_templates
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

-- ============================================================
-- 8. ADD RLS FOR SEO TABLES (migration 000015)
-- ============================================================

ALTER TABLE seo_projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_keywords ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_clusters ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_audits ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_internal_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_metadata ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_scores ENABLE ROW LEVEL SECURITY;

CREATE POLICY seo_projects_isolation ON seo_projects
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY seo_keywords_isolation ON seo_keywords
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM seo_projects sp WHERE sp.id = project_id
            AND sp.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY seo_clusters_isolation ON seo_clusters
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM seo_projects sp WHERE sp.id = project_id
            AND sp.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY seo_audits_isolation ON seo_audits
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY seo_internal_links_isolation ON seo_internal_links
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM seo_audits sa WHERE sa.id = audit_id
            AND sa.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY seo_metadata_isolation ON seo_metadata
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM seo_projects sp WHERE sp.id = project_id
            AND sp.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY seo_scores_isolation ON seo_scores
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );
