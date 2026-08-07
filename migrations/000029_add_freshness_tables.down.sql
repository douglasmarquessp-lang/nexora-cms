DROP TRIGGER IF EXISTS trg_freshness_content_updates_updated_at ON content_updates;
DROP TRIGGER IF EXISTS trg_freshness_news_intents_updated_at ON news_intents;

DROP POLICY IF EXISTS freshness_sweeps_isolation ON freshness_sweeps;
DROP POLICY IF EXISTS content_updates_isolation ON content_updates;
DROP POLICY IF EXISTS news_dedup_isolation ON news_dedup;
DROP POLICY IF EXISTS publication_versions_isolation ON publication_versions;
DROP POLICY IF EXISTS source_freshness_scores_isolation ON source_freshness_scores;
DROP POLICY IF EXISTS news_intents_isolation ON news_intents;

ALTER TABLE freshness_sweeps DISABLE ROW LEVEL SECURITY;
ALTER TABLE content_updates DISABLE ROW LEVEL SECURITY;
ALTER TABLE news_dedup DISABLE ROW LEVEL SECURITY;
ALTER TABLE publication_versions DISABLE ROW LEVEL SECURITY;
ALTER TABLE source_freshness_scores DISABLE ROW LEVEL SECURITY;
ALTER TABLE news_intents DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS freshness_sweeps;
DROP TABLE IF EXISTS content_updates;
DROP TABLE IF EXISTS news_dedup;
DROP TABLE IF EXISTS publication_versions;
DROP TABLE IF EXISTS source_freshness_scores;
DROP TABLE IF EXISTS news_intents;