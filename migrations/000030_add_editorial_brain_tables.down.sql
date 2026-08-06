-- AI Editorial Brain (Sprint 5.11) — rollback
DROP TRIGGER IF EXISTS trg_editorial_briefs_updated ON editorial_briefs;

DROP POLICY IF EXISTS editorial_evidence_isolation ON editorial_evidence;
DROP POLICY IF EXISTS editorial_block_scores_isolation ON editorial_block_scores;
DROP POLICY IF EXISTS editorial_reviews_isolation ON editorial_reviews;
DROP POLICY IF EXISTS editorial_briefs_isolation ON editorial_briefs;

ALTER TABLE editorial_evidence DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_block_scores DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_reviews DISABLE ROW LEVEL SECURITY;
ALTER TABLE editorial_briefs DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS editorial_evidence;
DROP TABLE IF EXISTS editorial_block_scores;
DROP TABLE IF EXISTS editorial_reviews;
DROP TABLE IF EXISTS editorial_briefs;
