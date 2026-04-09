-- name: SearchUnits :many
SELECT * FROM units
WHERE name_en ILIKE '%' || $1 || '%'
   OR name_de ILIKE '%' || $1 || '%'
   OR abbreviation_en ILIKE '%' || $1 || '%'
ORDER BY name_en LIMIT $2;

-- name: GetUnitByID :one
SELECT * FROM units WHERE id = $1;

-- name: CreateUnit :one
INSERT INTO units (name_en, name_de, abbreviation_en, abbreviation_de, plural_en, plural_de)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: UpdateUnit :one
UPDATE units SET
  name_en         = COALESCE(sqlc.narg('name_en'), name_en),
  name_de         = COALESCE(sqlc.narg('name_de'), name_de),
  abbreviation_en = COALESCE(sqlc.narg('abbreviation_en'), abbreviation_en),
  abbreviation_de = COALESCE(sqlc.narg('abbreviation_de'), abbreviation_de),
  plural_en       = COALESCE(sqlc.narg('plural_en'), plural_en),
  plural_de       = COALESCE(sqlc.narg('plural_de'), plural_de)
WHERE id = sqlc.arg('id') RETURNING *;

-- name: MergeUnitReferences :exec
UPDATE step_ingredients SET unit_id = $2 WHERE unit_id = $1;

-- name: DeleteUnit :exec
DELETE FROM units WHERE id = $1;

-- name: ListUnits :many
SELECT * FROM units ORDER BY name_en LIMIT $1 OFFSET $2;

-- name: CountUnits :one
SELECT COUNT(*) FROM units;
