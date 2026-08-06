-- 000027_add_translation_tables.down.sql
-- Reverse of 000027: drop RLS policies, triggers, and tables in FK-safe order.

ALTER TABLE translation_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE translation_stages DISABLE ROW LEVEL SECURITY;
ALTER TABLE glossary_terms DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS translation_jobs_isolation ON translation_jobs;
DROP POLICY IF EXISTS translation_stages_isolation ON translation_stages;
DROP POLICY IF EXISTS glossary_terms_isolation ON glossary_terms;

DROP TRIGGER IF EXISTS set_updated_at_translation_jobs ON translation_jobs;
DROP TRIGGER IF EXISTS set_updated_at_translation_stages ON translation_stages;
DROP TRIGGER IF EXISTS set_updated_at_glossary_terms ON glossary_terms;

DROP TABLE IF EXISTS translation_stages;
DROP TABLE IF EXISTS glossary_terms;
DROP TABLE IF EXISTS translation_jobs;
