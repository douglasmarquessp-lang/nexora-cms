DROP POLICY IF EXISTS research_fact_base_isolation ON research_fact_base;
DROP POLICY IF EXISTS research_cache_isolation ON research_cache;

ALTER TABLE research_fact_base DISABLE ROW LEVEL SECURITY;
ALTER TABLE research_cache DISABLE ROW LEVEL SECURITY;

ALTER TABLE research_sources
DROP COLUMN IF EXISTS reliability_score,
DROP COLUMN IF EXISTS domain;

DROP TABLE IF EXISTS research_fact_base;
DROP TABLE IF EXISTS research_cache;
