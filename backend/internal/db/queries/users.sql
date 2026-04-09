-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (username, email, display_name, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
  display_name = COALESCE(sqlc.narg('display_name'), display_name),
  email        = COALESCE(sqlc.narg('email'), email),
  password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
  updated_at   = NOW()
WHERE id = sqlc.arg('id') RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetSetupComplete :one
SELECT value FROM system_settings WHERE key = 'setup_complete';

-- name: SetSetupComplete :exec
INSERT INTO system_settings (key, value) VALUES ('setup_complete', 'true')
ON CONFLICT (key) DO UPDATE SET value = 'true', updated_at = NOW();
