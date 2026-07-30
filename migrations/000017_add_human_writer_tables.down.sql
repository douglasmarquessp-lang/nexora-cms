-- 000017_add_human_writer_tables.down.sql
-- Rollback human writer tables

DROP INDEX IF EXISTS idx_hh_created;
DROP INDEX IF EXISTS idx_hh_language;
DROP INDEX IF EXISTS idx_hh_profile;
DROP INDEX IF EXISTS idx_hh_site;
DROP TABLE IF EXISTS humanization_history CASCADE;

DROP INDEX IF EXISTS idx_st_language;
DROP INDEX IF EXISTS idx_st_category;
DROP INDEX IF EXISTS idx_st_profile;
DROP INDEX IF EXISTS idx_st_site;
DROP TABLE IF EXISTS sentence_templates CASCADE;

DROP INDEX IF EXISTS idx_sp_language;
DROP INDEX IF EXISTS idx_sp_type;
DROP INDEX IF EXISTS idx_sp_profile;
DROP INDEX IF EXISTS idx_sp_site;
DROP TABLE IF EXISTS style_patterns CASCADE;

DROP INDEX IF EXISTS idx_tl_formality;
DROP INDEX IF EXISTS idx_tl_language;
DROP INDEX IF EXISTS idx_tl_category;
DROP INDEX IF EXISTS idx_tl_site;
DROP TABLE IF EXISTS transition_library CASCADE;

DROP INDEX IF EXISTS idx_vs_language;
DROP INDEX IF EXISTS idx_vs_category;
DROP INDEX IF EXISTS idx_vs_site;
DROP TABLE IF EXISTS vocabulary_sets CASCADE;

DROP INDEX IF EXISTS idx_wp2_language;
DROP INDEX IF EXISTS idx_wp2_profile;
DROP INDEX IF EXISTS idx_wp2_site;
DROP TABLE IF EXISTS writing_personas CASCADE;

DROP INDEX IF EXISTS idx_wr_category;
DROP INDEX IF EXISTS idx_wr_rule_key;
DROP INDEX IF EXISTS idx_wr_profile;
DROP INDEX IF EXISTS idx_wr_site;
DROP TABLE IF EXISTS writing_rules CASCADE;

DROP INDEX IF EXISTS idx_wp_language;
DROP INDEX IF EXISTS idx_wp_slug;
DROP INDEX IF EXISTS idx_wp_site;
DROP TABLE IF EXISTS writing_profiles CASCADE;
