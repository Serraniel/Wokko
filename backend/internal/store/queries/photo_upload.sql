-- name: CreatePhotoUploadLink :exec
INSERT INTO photo_upload_links (token_hash, recipe_id, user_id, expires_at) VALUES (?, ?, ?, ?);

-- name: GetPhotoUploadLink :one
SELECT * FROM photo_upload_links WHERE token_hash = ?;

-- name: ClaimPhotoUploadLink :execrows
UPDATE photo_upload_links SET used_at = sqlc.arg(used_at) WHERE id = sqlc.arg(id) AND used_at IS NULL;

-- name: ReleasePhotoUploadLink :exec
UPDATE photo_upload_links SET used_at = NULL WHERE id = ?;

-- name: DeleteOldPhotoUploadLinks :exec
DELETE FROM photo_upload_links WHERE expires_at < ?;
