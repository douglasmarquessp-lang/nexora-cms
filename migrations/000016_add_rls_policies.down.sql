-- 000016_add_rls_policies.down.sql
-- Rollback RLS policy fixes and additions.
--
-- This migration:
--   1. Dropped buggy policies from 000005 and re-created them with correct site_id
--   2. Added RLS + policies for editorial, research, writer, editorial-engine,
--      generation, autocontent, and SEO tables.
--
-- The down migration reverses (2) completely and recreates the original buggy
-- policies from 000005 (using current_user_id instead of current_site_id) to
-- restore the pre-000016 state.

-- ============================================================
-- 8. DROP RLS POLICIES FOR SEO TABLES
-- ============================================================
DROP POLICY IF EXISTS seo_scores_isolation ON seo_scores;
DROP POLICY IF EXISTS seo_metadata_isolation ON seo_metadata;
DROP POLICY IF EXISTS seo_internal_links_isolation ON seo_internal_links;
DROP POLICY IF EXISTS seo_audits_isolation ON seo_audits;
DROP POLICY IF EXISTS seo_clusters_isolation ON seo_clusters;
DROP POLICY IF EXISTS seo_keywords_isolation ON seo_keywords;
DROP POLICY IF EXISTS seo_projects_isolation ON seo_projects;

ALTER TABLE seo_scores DISABLE ROW LEVEL SECURITY;
ALTER TABLE seo_metadata DISABLE ROW LEVEL SECURITY;
ALTER TABLE seo_internal_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE seo_audits DISABLE ROW LEVEL SECURITY;
ALTER TABLE seo_clusters DISABLE ROW LEVEL SECURITY;
ALTER TABLE seo_keywords DISABLE ROW LEVEL SECURITY;
ALTER TABLE seo_projects DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 7. DROP RLS POLICIES FOR AUTOCONTENT TABLES
-- ============================================================
DROP POLICY IF EXISTS workflow_templates_isolation ON workflow_templates;
DROP POLICY IF EXISTS autocontent_queue_isolation ON autocontent_queue;
DROP POLICY IF EXISTS autocontent_results_isolation ON autocontent_results;
DROP POLICY IF EXISTS autocontent_steps_isolation ON autocontent_steps;
DROP POLICY IF EXISTS autocontent_jobs_isolation ON autocontent_jobs;

ALTER TABLE workflow_templates DISABLE ROW LEVEL SECURITY;
ALTER TABLE autocontent_queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE autocontent_results DISABLE ROW LEVEL SECURITY;
ALTER TABLE autocontent_steps DISABLE ROW LEVEL SECURITY;
ALTER TABLE autocontent_jobs DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 6. DROP RLS POLICIES FOR GENERATION TABLES
-- ============================================================
DROP POLICY IF EXISTS generation_stats_isolation ON generation_stats;
DROP POLICY IF EXISTS generation_quality_gates_isolation ON generation_quality_gates;
DROP POLICY IF EXISTS generation_pipeline_logs_isolation ON generation_pipeline_logs;
DROP POLICY IF EXISTS generation_pipeline_isolation ON generation_pipeline;
DROP POLICY IF EXISTS generation_jobs_isolation ON generation_jobs;

ALTER TABLE generation_stats DISABLE ROW LEVEL SECURITY;
ALTER TABLE generation_quality_gates DISABLE ROW LEVEL SECURITY;
ALTER TABLE generation_pipeline_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE generation_pipeline DISABLE ROW LEVEL SECURITY;
ALTER TABLE generation_jobs DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 5. DROP RLS POLICIES FOR EDITORIAL ENGINE TABLES
-- ============================================================
DROP POLICY IF EXISTS editorial_prompt_data_isolation ON editorial_prompt_data;
DROP POLICY IF EXISTS editorial_translations_isolation ON editorial_translations;
DROP POLICY IF EXISTS editorial_quality_scores_isolation ON editorial_quality_scores;
DROP POLICY IF EXISTS editorial_seo_data_isolation ON editorial_seo_data;
DROP POLICY IF EXISTS editorial_style_rules_isolation ON editorial_style_rules;
DROP POLICY IF EXISTS pipeline_stages_isolation ON pipeline_stages;
DROP POLICY IF EXISTS editorial_pipelines_isolation ON editorial_pipelines;

ALTER TABLE editorial_prompt_data DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_translations DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_quality_scores DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_seo_data DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_style_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE pipeline_stages DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_pipelines DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 4. DROP RLS POLICIES FOR WRITER TABLES
-- ============================================================
DROP POLICY IF EXISTS article_versions_isolation ON article_versions;
DROP POLICY IF EXISTS article_sections_isolation ON article_sections;
DROP POLICY IF EXISTS article_outlines_isolation ON article_outlines;
DROP POLICY IF EXISTS article_jobs_isolation ON article_jobs;
DROP POLICY IF EXISTS writing_styles_isolation ON writing_styles;

ALTER TABLE article_versions DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_sections DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_outlines DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE writing_styles DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 3. DROP RLS POLICIES FOR RESEARCH TABLES
-- ============================================================
DROP POLICY IF EXISTS research_briefings_isolation ON research_briefings;
DROP POLICY IF EXISTS research_entities_isolation ON research_entities;
DROP POLICY IF EXISTS research_sources_isolation ON research_sources;
DROP POLICY IF EXISTS research_jobs_isolation ON research_jobs;

ALTER TABLE research_briefings DISABLE ROW LEVEL SECURITY;
ALTER TABLE research_entities DISABLE ROW LEVEL SECURITY;
ALTER TABLE research_sources DISABLE ROW LEVEL SECURITY;
ALTER TABLE research_jobs DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 2. DROP RLS POLICIES FOR EDITORIAL TABLES
-- ============================================================
DROP POLICY IF EXISTS editorial_widgets_isolation ON editorial_widgets;
DROP POLICY IF EXISTS editorial_calendar_events_isolation ON editorial_calendar_events;
DROP POLICY IF EXISTS approval_requests_isolation ON approval_requests;
DROP POLICY IF EXISTS post_revisions_isolation ON post_revisions;
DROP POLICY IF EXISTS editorial_tasks_isolation ON editorial_tasks;

ALTER TABLE editorial_widgets DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_calendar_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE approval_requests DISABLE ROW LEVEL SECURITY;
ALTER TABLE post_revisions DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_tasks DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 1. RESTORE ORIGINAL (BUGGY) RLS POLICIES FROM 000005
--    These use current_user_id instead of current_site_id for site_id
--    comparison, which is the bug that 000016 fixed.
-- ============================================================
DROP POLICY IF EXISTS tags_isolation ON tags;
DROP POLICY IF EXISTS categories_isolation ON categories;
DROP POLICY IF EXISTS posts_isolation ON posts;

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
