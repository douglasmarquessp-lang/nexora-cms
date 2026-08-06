-- Editorial Pipeline board (Sprint 5.12)
-- posts.scheduled_at already exists (000005); this adds the missing
-- site-scoped index for the "Agendado" column ordering.

CREATE INDEX IF NOT EXISTS idx_posts_site_scheduled
    ON posts(site_id, scheduled_at)
    WHERE status = 'scheduled' AND deleted_at IS NULL;
