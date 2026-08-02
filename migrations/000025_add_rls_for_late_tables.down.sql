-- 000025_add_rls_for_late_tables.down.sql
-- Rollback RLS added for tables created after migration 000016.

-- ============================================================
-- 5. ARTICLE SOURCES (migration 000023)
-- ============================================================

DROP POLICY IF EXISTS article_sources_isolation ON article_sources;
ALTER TABLE article_sources DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 4. WORKFLOW TABLES (migration 000021)
-- ============================================================

DROP POLICY IF EXISTS workflow_dashboard_isolation ON workflow_dashboard;
DROP POLICY IF EXISTS workflow_notifications_isolation ON workflow_notifications;
DROP POLICY IF EXISTS workflow_history_isolation ON workflow_history;
DROP POLICY IF EXISTS workflow_queue_isolation ON workflow_queue;
DROP POLICY IF EXISTS workflow_steps_isolation ON workflow_steps;
DROP POLICY IF EXISTS workflow_jobs_isolation ON workflow_jobs;

ALTER TABLE workflow_dashboard DISABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_notifications DISABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_steps DISABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_jobs DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 3. PUBLISHER TABLES (migration 000019)
-- ============================================================

DROP POLICY IF EXISTS publication_metrics_isolation ON publication_metrics;
DROP POLICY IF EXISTS publication_schedule_isolation ON publication_schedule;
DROP POLICY IF EXISTS publication_queue_isolation ON publication_queue;
DROP POLICY IF EXISTS publication_history_isolation ON publication_history;
DROP POLICY IF EXISTS publications_isolation ON publications;

ALTER TABLE publication_metrics DISABLE ROW LEVEL SECURITY;
ALTER TABLE publication_schedule DISABLE ROW LEVEL SECURITY;
ALTER TABLE publication_queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE publication_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE publications DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 2. ARTICLE PIPELINE TABLES (migration 000018)
-- ============================================================

DROP POLICY IF EXISTS publication_candidates_isolation ON publication_candidates;
DROP POLICY IF EXISTS article_quality_reports_isolation ON article_quality_reports;
DROP POLICY IF EXISTS article_pipeline_metrics_isolation ON article_pipeline_metrics;
DROP POLICY IF EXISTS article_pipeline_steps_isolation ON article_pipeline_steps;
DROP POLICY IF EXISTS article_pipeline_jobs_isolation ON article_pipeline_jobs;

ALTER TABLE publication_candidates DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_quality_reports DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_pipeline_metrics DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_pipeline_steps DISABLE ROW LEVEL SECURITY;
ALTER TABLE article_pipeline_jobs DISABLE ROW LEVEL SECURITY;

-- ============================================================
-- 1. HUMAN WRITER TABLES (migration 000017)
-- ============================================================

DROP POLICY IF EXISTS humanization_history_isolation ON humanization_history;
DROP POLICY IF EXISTS sentence_templates_isolation ON sentence_templates;
DROP POLICY IF EXISTS style_patterns_isolation ON style_patterns;
DROP POLICY IF EXISTS transition_library_isolation ON transition_library;
DROP POLICY IF EXISTS vocabulary_sets_isolation ON vocabulary_sets;
DROP POLICY IF EXISTS writing_personas_isolation ON writing_personas;
DROP POLICY IF EXISTS writing_rules_isolation ON writing_rules;
DROP POLICY IF EXISTS writing_profiles_isolation ON writing_profiles;

ALTER TABLE humanization_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE sentence_templates DISABLE ROW LEVEL SECURITY;
ALTER TABLE style_patterns DISABLE ROW LEVEL SECURITY;
ALTER TABLE transition_library DISABLE ROW LEVEL SECURITY;
ALTER TABLE vocabulary_sets DISABLE ROW LEVEL SECURITY;
ALTER TABLE writing_personas DISABLE ROW LEVEL SECURITY;
ALTER TABLE writing_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE writing_profiles DISABLE ROW LEVEL SECURITY;
