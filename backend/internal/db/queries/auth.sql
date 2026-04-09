-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 AND expires_at > NOW();

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token_hash = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = $1;

-- name: CreateInviteLink :one
INSERT INTO invite_links (token, created_by, max_uses, expires_at)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetInviteLinkByToken :one
SELECT * FROM invite_links WHERE token = $1;

-- name: UseInviteLink :exec
UPDATE invite_links SET use_count = use_count + 1, used_by = $2
WHERE token = $1;

-- name: ListInviteLinks :many
SELECT * FROM invite_links ORDER BY created_at DESC;

-- name: DeleteInviteLink :exec
DELETE FROM invite_links WHERE id = $1;

-- name: CreatePasskeyCredential :one
INSERT INTO passkey_credentials (user_id, credential_id, public_key, sign_count, name)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetPasskeyCredentialByCredentialID :one
SELECT * FROM passkey_credentials WHERE credential_id = $1;

-- name: UpdatePasskeySignCount :exec
UPDATE passkey_credentials SET sign_count = $2 WHERE credential_id = $1;

-- name: ListUserPasskeys :many
SELECT * FROM passkey_credentials WHERE user_id = $1;

-- name: DeletePasskeyCredential :exec
DELETE FROM passkey_credentials WHERE id = $1 AND user_id = $2;
