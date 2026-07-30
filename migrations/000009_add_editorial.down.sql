-- 000009_add_editorial.down.sql
-- Rollback editorial task management tables and seed data

-- Seed data (delete the auto-inserted widgets)
DELETE FROM editorial_widgets WHERE widget_type IN (
    'stats', 'recent_posts', 'draft_posts', 'scheduled_posts', 'tasks', 'calendar'
);

DROP INDEX IF EXISTS idx_editorial_widgets_site_id;
DROP TABLE IF EXISTS editorial_widgets CASCADE;

DROP INDEX IF EXISTS idx_calendar_events_site_date;
DROP TABLE IF EXISTS editorial_calendar_events CASCADE;

DROP INDEX IF EXISTS idx_approval_requests_status;
DROP INDEX IF EXISTS idx_approval_requests_post_id;
DROP TABLE IF EXISTS approval_requests CASCADE;

DROP INDEX IF EXISTS idx_post_revisions_version;
DROP INDEX IF EXISTS idx_post_revisions_site_id;
DROP INDEX IF EXISTS idx_post_revisions_post_id;
DROP TABLE IF EXISTS post_revisions CASCADE;

DROP INDEX IF EXISTS idx_editorial_tasks_due_date;
DROP INDEX IF EXISTS idx_editorial_tasks_status;
DROP INDEX IF EXISTS idx_editorial_tasks_site_id;
DROP TABLE IF EXISTS editorial_tasks CASCADE;
