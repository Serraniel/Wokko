-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: CreateHousehold :one
INSERT INTO households (name) VALUES (?) RETURNING *;

-- name: GetHousehold :one
SELECT * FROM households WHERE id = ?;

-- name: RenameHousehold :exec
UPDATE households SET name = ? WHERE id = ?;

-- name: CountHouseholdMembers :one
SELECT count(*) FROM users WHERE household_id = ?;

-- name: DeleteHousehold :exec
DELETE FROM households WHERE id = ?;

-- name: CreateUser :one
INSERT INTO users (handle, display_name, password_hash, role, locale, household_id)
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByHandle :one
SELECT * FROM users WHERE handle = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY handle;

-- name: ListHouseholdMembers :many
SELECT * FROM users WHERE household_id = ? ORDER BY handle;

-- name: UpdateProfile :exec
UPDATE users SET display_name = ?, locale = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: SetUserDisabled :exec
UPDATE users SET disabled_at = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: SetUserRole :exec
UPDATE users SET role = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: SetUserHousehold :exec
UPDATE users SET household_id = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CountActiveAdmins :one
SELECT count(*) FROM users WHERE role = 'admin' AND disabled_at IS NULL;

-- name: CreateSession :exec
INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?);

-- name: GetSessionUser :one
SELECT sqlc.embed(sessions), sqlc.embed(users) FROM sessions JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = ? AND sessions.expires_at > ? AND users.disabled_at IS NULL;

-- name: ExtendSession :exec
UPDATE sessions SET expires_at = ? WHERE id = ?;

-- name: DeleteSessionByHash :exec
DELETE FROM sessions WHERE token_hash = ?;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= ?;

-- name: CreateInvite :one
INSERT INTO invites (token_hash, created_by, household_id, max_uses, expires_at)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: GetInviteByHash :one
SELECT * FROM invites WHERE token_hash = ?;

-- name: ListRegistrationInvites :many
SELECT * FROM invites WHERE household_id IS NULL ORDER BY created_at DESC, id DESC;

-- name: ListHouseholdInvites :many
SELECT * FROM invites WHERE household_id = ? ORDER BY created_at DESC, id DESC;

-- name: UseInvite :execrows
UPDATE invites SET use_count = use_count + 1 WHERE id = ? AND use_count < max_uses;

-- name: DeleteInvite :exec
DELETE FROM invites WHERE id = ?;

-- name: GetInvite :one
SELECT * FROM invites WHERE id = ?;

-- name: GetSetting :one
SELECT value FROM settings WHERE key = ?;

-- name: SetSetting :exec
INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value;

-- name: CreateFriendship :exec
INSERT INTO friendships (requester_id, addressee_id) VALUES (?, ?) ON CONFLICT DO NOTHING;

-- name: GetFriendshipBetween :one
SELECT * FROM friendships
WHERE (requester_id = sqlc.arg(a) AND addressee_id = sqlc.arg(b)) OR (requester_id = sqlc.arg(b) AND addressee_id = sqlc.arg(a));

-- name: AcceptFriendship :execrows
UPDATE friendships SET accepted_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ? AND addressee_id = ? AND accepted_at IS NULL;

-- name: DeleteFriendship :execrows
DELETE FROM friendships WHERE id = ? AND (requester_id = sqlc.arg(user_id) OR addressee_id = sqlc.arg(user_id));

-- name: ListFriendships :many
SELECT f.id, f.requester_id, f.addressee_id, f.accepted_at, f.created_at,
       u.id AS other_id, u.handle AS other_handle, u.display_name AS other_display_name
FROM friendships f
JOIN users u ON u.id = CASE WHEN f.requester_id = sqlc.arg(user_id) THEN f.addressee_id ELSE f.requester_id END
WHERE f.requester_id = sqlc.arg(user_id) OR f.addressee_id = sqlc.arg(user_id)
ORDER BY u.handle;

-- name: ListVisibleUsers :many
SELECT u.* FROM users u WHERE u.id = sqlc.arg(user_id) OR u.household_id = sqlc.arg(household_id)
   OR u.id IN (SELECT addressee_id FROM friendships WHERE requester_id = sqlc.arg(user_id) AND accepted_at IS NOT NULL)
   OR u.id IN (SELECT requester_id FROM friendships WHERE addressee_id = sqlc.arg(user_id) AND accepted_at IS NOT NULL)
ORDER BY u.handle;
