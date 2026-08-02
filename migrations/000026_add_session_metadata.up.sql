-- 000026_add_session_metadata.up.sql
-- Add missing session metadata columns (device_info, ip_address)
-- required by the auth service session creation (INSERT/RETURNING).

ALTER TABLE sessions
ADD COLUMN IF NOT EXISTS device_info TEXT DEFAULT '';

ALTER TABLE sessions
ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45) DEFAULT '';
