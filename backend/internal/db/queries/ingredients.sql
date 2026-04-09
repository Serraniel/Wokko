-- name: SearchIngredients :many
SELECT * FROM ingredients
WHERE name_en ILIKE '%' || $1 || '%'
   OR name_de ILIKE '%' || $1 || '%'
ORDER BY name_en LIMIT $2;

-- name: GetIngredientByID :one
SELECT * FROM ingredients WHERE id = $1;

-- name: GetIngredientByNameEN :one
SELECT * FROM ingredients WHERE LOWER(name_en) = LOWER($1) LIMIT 1;

-- name: CreateIngredient :one
INSERT INTO ingredients (name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateIngredient :one
UPDATE ingredients SET
  name_en         = COALESCE(sqlc.narg('name_en'), name_en),
  name_de         = COALESCE(sqlc.narg('name_de'), name_de),
  plural_en       = COALESCE(sqlc.narg('plural_en'), plural_en),
  plural_de       = COALESCE(sqlc.narg('plural_de'), plural_de),
  default_unit_id = COALESCE(sqlc.narg('default_unit_id'), default_unit_id)
WHERE id = sqlc.arg('id') RETURNING *;

-- name: MergeIngredientReferences :exec
UPDATE step_ingredients SET ingredient_id = $2 WHERE ingredient_id = $1;

-- name: DeleteIngredient :exec
DELETE FROM ingredients WHERE id = $1;

-- name: ListIngredients :many
SELECT * FROM ingredients ORDER BY name_en LIMIT $1 OFFSET $2;

-- name: CountIngredients :one
SELECT COUNT(*) FROM ingredients;

-- name: GetSuggestedUnitForIngredient :one
SELECT unit_id FROM ingredient_unit_stats
WHERE ingredient_id = $1 ORDER BY use_count DESC LIMIT 1;

-- name: IncrementIngredientUnitStat :exec
INSERT INTO ingredient_unit_stats (ingredient_id, unit_id, use_count)
VALUES ($1, $2, 1)
ON CONFLICT (ingredient_id, unit_id)
DO UPDATE SET use_count = ingredient_unit_stats.use_count + 1;
