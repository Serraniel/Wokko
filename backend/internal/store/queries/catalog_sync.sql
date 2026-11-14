-- name: LastCatalogSync :one
SELECT * FROM catalog_syncs ORDER BY id DESC LIMIT 1;

-- name: AddCatalogSync :exec
INSERT INTO catalog_syncs (pack, version, added, updated, suggested) VALUES (?, ?, ?, ?, ?);

-- name: GetCatalogEntry :one
SELECT * FROM catalog_entries WHERE kind = ? AND key = ?;

-- name: ListCatalogEntries :many
SELECT * FROM catalog_entries WHERE kind = ?;

-- name: SaveCatalogEntry :exec
INSERT INTO catalog_entries (kind, key, entity_id, base) VALUES (?, ?, ?, ?)
ON CONFLICT (kind, key) DO UPDATE SET entity_id = excluded.entity_id, base = excluded.base;

-- name: CatalogEntityIsEntry :one
SELECT count(*) FROM catalog_entries WHERE kind = ? AND entity_id = ?;

-- name: ForgetCatalogEntity :exec
UPDATE catalog_entries SET entity_id = NULL WHERE kind = ? AND entity_id = ?;

-- name: ListCatalogSuggestions :many
SELECT * FROM catalog_suggestions ORDER BY kind, key, field;

-- name: GetCatalogSuggestion :one
SELECT * FROM catalog_suggestions WHERE id = ?;

-- name: ClearCatalogSuggestions :exec
DELETE FROM catalog_suggestions WHERE kind = ? AND key = ?;

-- name: AddCatalogSuggestion :exec
INSERT INTO catalog_suggestions (kind, key, entity_id, field, ours, theirs) VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteCatalogSuggestion :exec
DELETE FROM catalog_suggestions WHERE id = ?;

-- name: SetUnitGrams :exec
UPDATE units SET grams = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: GetFoodOwner :one
SELECT created_by FROM foods WHERE id = ?;

-- name: GetUnitOwner :one
SELECT created_by FROM units WHERE id = ?;

-- name: CreateCatalogFood :one
INSERT INTO foods (created_by) VALUES (NULL) RETURNING id;

-- name: CreateCatalogUnit :one
INSERT INTO units (created_by) VALUES (NULL) RETURNING id;
