-- 000026_add_session_metadata.down.sql

ALTER TABLE sessions
DROP COLUMN IF EXISTS device_info;

ALTER TABLE sessions
DROP COLUMN IF EXISTS ip_address;
