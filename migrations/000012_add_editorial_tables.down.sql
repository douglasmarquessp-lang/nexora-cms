-- 000012_add_editorial_tables.down.sql
-- Rollback editorial pipeline, style rules, seo data, translations

DROP INDEX IF EXISTS idx_prompt_data_site;
DROP INDEX IF EXISTS idx_prompt_data_job;
DROP TABLE IF EXISTS editorial_prompt_data CASCADE;

DROP INDEX IF EXISTS idx_translations_status;
DROP INDEX IF EXISTS idx_translations_site;
DROP INDEX IF EXISTS idx_translations_job;
DROP TABLE IF EXISTS editorial_translations CASCADE;

DROP INDEX IF EXISTS idx_quality_scores_site;
DROP INDEX IF EXISTS idx_quality_scores_job;
DROP TABLE IF EXISTS editorial_quality_scores CASCADE;

DROP INDEX IF EXISTS idx_seo_data_site;
DROP INDEX IF EXISTS idx_seo_data_job;
DROP TABLE IF EXISTS editorial_seo_data CASCADE;

DROP INDEX IF EXISTS idx_style_rules_site;
DROP TABLE IF EXISTS editorial_style_rules CASCADE;

DROP INDEX IF EXISTS idx_pipeline_stages_status;
DROP INDEX IF EXISTS idx_pipeline_stages_stage;
DROP INDEX IF EXISTS idx_pipeline_stages_pipeline;
DROP TABLE IF EXISTS pipeline_stages CASCADE;

DROP INDEX IF EXISTS idx_editorial_pipelines_status;
DROP INDEX IF EXISTS idx_editorial_pipelines_stage;
DROP INDEX IF EXISTS idx_editorial_pipelines_site;
DROP INDEX IF EXISTS idx_editorial_pipelines_job;
DROP TABLE IF EXISTS editorial_pipelines CASCADE;
