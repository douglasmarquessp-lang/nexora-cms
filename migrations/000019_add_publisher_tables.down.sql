-- 000019_add_publisher_tables.down.sql
-- Rollback publisher tables.
--
-- NOTE: publication_queue is owned by this migration (not 000014).
-- The historical conflict where 000014 originally created publication_queue
-- was resolved in Sprint 3.7 by renaming it to autocontent_queue.

DROP INDEX IF EXISTS idx_pub_metrics_site;
DROP INDEX IF EXISTS idx_pub_metrics_publication;
DROP TABLE IF EXISTS publication_metrics CASCADE;

DROP INDEX IF EXISTS idx_pub_schedule_status;
DROP INDEX IF EXISTS idx_pub_schedule_at;
DROP INDEX IF EXISTS idx_pub_schedule_publication;
DROP INDEX IF EXISTS idx_pub_schedule_site;
DROP TABLE IF EXISTS publication_schedule CASCADE;

DROP INDEX IF EXISTS idx_pub_queue_scheduled;
DROP INDEX IF EXISTS idx_pub_queue_status;
DROP INDEX IF EXISTS idx_pub_queue_site;
DROP TABLE IF EXISTS publication_queue CASCADE;

DROP INDEX IF EXISTS idx_pub_history_action;
DROP INDEX IF EXISTS idx_pub_history_site;
DROP INDEX IF EXISTS idx_pub_history_publication;
DROP TABLE IF EXISTS publication_history CASCADE;

DROP INDEX IF EXISTS idx_pub_unique_slug;
DROP INDEX IF EXISTS idx_pub_author;
DROP INDEX IF EXISTS idx_pub_scheduled_at;
DROP INDEX IF EXISTS idx_pub_published_at;
DROP INDEX IF EXISTS idx_pub_language;
DROP INDEX IF EXISTS idx_pub_status;
DROP INDEX IF EXISTS idx_pub_slug;
DROP INDEX IF EXISTS idx_pub_site;
DROP TABLE IF EXISTS publications CASCADE;
