-- name: FindUserByEmail :one
SELECT id, email, password_hash, name, avatar, role, metadata, mfa_enabled, mfa_secret, last_login, created_at, updated_at
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: FindUserByID :one
SELECT id, email, password_hash, name, avatar, role, metadata, mfa_enabled, mfa_secret, last_login, created_at, updated_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, role, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, password_hash, name, avatar, role, metadata, mfa_enabled, mfa_secret, last_login, created_at, updated_at;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, refresh_token, device_info, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, refresh_token, device_info, ip_address, expires_at, created_at;

-- name: FindSessionByRefreshToken :one
SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, created_at
FROM sessions
WHERE refresh_token = $1 AND expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: CreateOAuthAccount :one
INSERT INTO oauth_accounts (user_id, provider, provider_id, access_token, refresh_token, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, provider, provider_id, access_token, refresh_token, expires_at, created_at;

-- name: FindOAuthAccount :one
SELECT id, user_id, provider, provider_id, access_token, refresh_token, expires_at, created_at
FROM oauth_accounts
WHERE provider = $1 AND provider_id = $2;

-- name: FindOAuthAccountsByUserID :many
SELECT id, user_id, provider, provider_id, access_token, refresh_token, expires_at, created_at
FROM oauth_accounts
WHERE user_id = $1;

-- name: LinkOAuthToUser :exec
UPDATE oauth_accounts SET user_id = $1, updated_at = NOW()
WHERE provider = $2 AND provider_id = $3;

-- name: UpsertMFAConfig :one
INSERT INTO mfa_configs (user_id, secret, enabled, method, backup_codes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
    secret = EXCLUDED.secret,
    enabled = EXCLUDED.enabled,
    method = EXCLUDED.method,
    backup_codes = EXCLUDED.backup_codes,
    updated_at = NOW()
RETURNING id, user_id, secret, enabled, method, backup_codes, created_at, updated_at;

-- name: FindMFAConfigByUserID :one
SELECT id, user_id, secret, enabled, method, backup_codes, created_at, updated_at
FROM mfa_configs
WHERE user_id = $1;

-- name: DisableMFA :exec
UPDATE mfa_configs SET enabled = false, updated_at = NOW()
WHERE user_id = $1;
