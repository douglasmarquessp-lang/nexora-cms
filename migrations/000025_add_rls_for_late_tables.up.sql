-- 000025_add_rls_for_late_tables.up.sql
-- Complete RLS coverage for tables created AFTER migration 000016
-- (which added RLS for all tables that existed at that time).
--
-- Tables covered, by origin migration:
--   000017 human writer: writing_profiles, writing_rules, writing_personas,
--           vocabulary_sets, transition_library, style_patterns,
--           sentence_templates, humanization_history
--   000018 article pipeline: article_pipeline_jobs, article_pipeline_steps,
--           article_pipeline_metrics, article_quality_reports,
--           publication_candidates
--   000019 publisher: publications, publication_history, publication_queue,
--           publication_schedule, publication_metrics
--   000021 workflow: workflow_jobs, workflow_steps, workflow_queue,
--           workflow_history, workflow_notifications, workflow_dashboard
--   000023 grounding: article_sources
--
-- Excluded: system_installation (000022) has no site_id column — it is a
-- global single-row table, so per-site isolation does not apply.
-- plugin tables (000008) have no site_id either (global, pre-000016).

-- ============================================================
-- 1. HUMAN WRITER TABLES (migration 000017)
-- ============================================================

ALTER TABLE writing_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE writing_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE writing_personas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vocabulary_sets ENABLE ROW LEVEL SECURITY;
ALTER TABLE transition_library ENABLE ROW LEVEL SECURITY;
ALTER TABLE style_patterns ENABLE ROW LEVEL SECURITY;
ALTER TABLE sentence_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE humanization_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY writing_profiles_isolation ON writing_profiles
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY writing_rules_isolation ON writing_rules
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY writing_personas_isolation ON writing_personas
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY vocabulary_sets_isolation ON vocabulary_sets
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY transition_library_isolation ON transition_library
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY style_patterns_isolation ON style_patterns
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY sentence_templates_isolation ON sentence_templates
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY humanization_history_isolation ON humanization_history
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
-- 2. ARTICLE PIPELINE TABLES (migration 000018)
-- ============================================================

ALTER TABLE article_pipeline_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_pipeline_steps ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_pipeline_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE article_quality_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE publication_candidates ENABLE ROW LEVEL SECURITY;

CREATE POLICY article_pipeline_jobs_isolation ON article_pipeline_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY article_pipeline_steps_isolation ON article_pipeline_steps
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_pipeline_jobs aj WHERE aj.id = pipeline_job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY article_pipeline_metrics_isolation ON article_pipeline_metrics
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_pipeline_jobs aj WHERE aj.id = pipeline_job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY article_quality_reports_isolation ON article_quality_reports
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM article_pipeline_jobs aj WHERE aj.id = pipeline_job_id
            AND aj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY publication_candidates_isolation ON publication_candidates
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
-- 3. PUBLISHER TABLES (migration 000019)
--    publication_queue is owned by 000019 (publisher module).
-- ============================================================

ALTER TABLE publications ENABLE ROW LEVEL SECURITY;
ALTER TABLE publication_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE publication_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE publication_schedule ENABLE ROW LEVEL SECURITY;
ALTER TABLE publication_metrics ENABLE ROW LEVEL SECURITY;

CREATE POLICY publications_isolation ON publications
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY publication_history_isolation ON publication_history
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
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

CREATE POLICY publication_schedule_isolation ON publication_schedule
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY publication_metrics_isolation ON publication_metrics
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
-- 4. WORKFLOW TABLES (migration 000021)
-- ============================================================

ALTER TABLE workflow_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_steps ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_dashboard ENABLE ROW LEVEL SECURITY;

CREATE POLICY workflow_jobs_isolation ON workflow_jobs
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY workflow_steps_isolation ON workflow_steps
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM workflow_jobs wj WHERE wj.id = workflow_job_id
            AND wj.site_id = current_setting('app.current_site_id')::UUID
        )
        OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin')
    );

CREATE POLICY workflow_queue_isolation ON workflow_queue
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY workflow_history_isolation ON workflow_history
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY workflow_notifications_isolation ON workflow_notifications
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );

CREATE POLICY workflow_dashboard_isolation ON workflow_dashboard
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
-- 5. ARTICLE SOURCES (migration 000023)
-- ============================================================

ALTER TABLE article_sources ENABLE ROW LEVEL SECURITY;

CREATE POLICY article_sources_isolation ON article_sources
    FOR ALL
    USING (
        site_id = current_setting('app.current_site_id')::UUID
        OR EXISTS (
            SELECT 1 FROM sites WHERE id = site_id
            AND (owner_id = current_setting('app.current_user_id')::UUID
                 OR current_setting('app.current_user_role') IN ('superadmin', 'siteadmin'))
        )
    );
