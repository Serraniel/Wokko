-- name: CreatePasskey :one
INSERT INTO passkeys (user_id, name, credential_id, credential) VALUES (?, ?, ?, ?) RETURNING *;

-- name: ListPasskeys :many
SELECT * FROM passkeys WHERE user_id = ? ORDER BY id;

-- name: UpdatePasskeyUse :exec
UPDATE passkeys SET credential = ?, last_used_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE credential_id = ?;

-- name: DeletePasskey :execrows
DELETE FROM passkeys WHERE id = ? AND user_id = ?;

-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, scope, client_id, expires_at) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetAPITokenByHash :one
SELECT * FROM api_tokens WHERE token_hash = ?;

-- name: ListAPITokens :many
SELECT * FROM api_tokens WHERE user_id = ? ORDER BY id DESC;

-- name: DeleteAPIToken :execrows
DELETE FROM api_tokens WHERE id = ? AND user_id = ?;

-- name: DeleteAPITokensForClient :exec
DELETE FROM api_tokens WHERE user_id = ? AND client_id = ?;

-- name: TouchAPIToken :exec
UPDATE api_tokens SET last_used_at = ? WHERE id = ?;

-- name: DeleteExpiredAPITokens :exec
DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at <= ?;

-- name: CreateOAuthClient :exec
INSERT INTO oauth_clients (client_id, name, redirect_uris) VALUES (?, ?, ?);

-- name: GetOAuthClient :one
SELECT * FROM oauth_clients WHERE client_id = ?;

-- name: CreateOAuthCode :exec
INSERT INTO oauth_codes (code_hash, client_id, user_id, redirect_uri, challenge, scope, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetOAuthCode :one
SELECT * FROM oauth_codes WHERE code_hash = ?;

-- name: DeleteOAuthCode :exec
DELETE FROM oauth_codes WHERE code_hash = ?;

-- name: DeleteExpiredOAuthCodes :exec
DELETE FROM oauth_codes WHERE expires_at <= ?;

-- name: CreateOAuthRefresh :exec
INSERT INTO oauth_refresh_tokens (token_hash, client_id, user_id, scope, expires_at) VALUES (?, ?, ?, ?, ?);

-- name: GetOAuthRefresh :one
SELECT * FROM oauth_refresh_tokens WHERE token_hash = ?;

-- name: DeleteOAuthRefresh :exec
DELETE FROM oauth_refresh_tokens WHERE token_hash = ?;

-- name: DeleteOAuthRefreshForClient :exec
DELETE FROM oauth_refresh_tokens WHERE user_id = ? AND client_id = ?;

-- name: AddAssistantActivity :exec
INSERT INTO assistant_activity (user_id, token_name, tool, target) VALUES (?, ?, ?, ?);

-- name: ListAssistantActivity :many
SELECT * FROM assistant_activity WHERE user_id = ? ORDER BY id DESC LIMIT 100;

-- name: MaxAssistantActivityID :one
SELECT CAST(COALESCE(MAX(id), 0) AS INTEGER) FROM assistant_activity WHERE user_id = ?;

-- name: DeleteOldAssistantActivity :exec
DELETE FROM assistant_activity WHERE user_id = ? AND id <= ?;
