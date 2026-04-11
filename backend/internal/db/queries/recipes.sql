-- name: CreateRecipe :one
INSERT INTO recipes (author_id, name, description, photo_path, work_time_min, servings, is_public, source_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetRecipeByID :one
SELECT * FROM recipes WHERE id = $1;

-- name: ListRecipesByAuthor :many
SELECT * FROM recipes WHERE author_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: ListPublicRecipes :many
SELECT * FROM recipes WHERE is_public = true ORDER BY updated_at DESC LIMIT $1 OFFSET $2;

-- name: CountPublicRecipes :one
SELECT COUNT(*) FROM recipes WHERE is_public = true;

-- name: CountRecipesByAuthor :one
SELECT COUNT(*) FROM recipes WHERE author_id = $1;

-- name: UpdateRecipe :one
UPDATE recipes SET
  name          = COALESCE(sqlc.narg('name'), name),
  description   = COALESCE(sqlc.narg('description'), description),
  photo_path    = COALESCE(sqlc.narg('photo_path'), photo_path),
  work_time_min = COALESCE(sqlc.narg('work_time_min'), work_time_min),
  servings      = COALESCE(sqlc.narg('servings'), servings),
  is_public     = COALESCE(sqlc.narg('is_public'), is_public),
  source_url    = COALESCE(sqlc.narg('source_url'), source_url),
  updated_at    = NOW()
WHERE id = sqlc.arg('id') RETURNING *;

-- name: DeleteRecipe :exec
DELETE FROM recipes WHERE id = $1;

-- name: CreateRecipeStep :one
INSERT INTO recipe_steps (recipe_id, step_order, description, duration_min)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetRecipeStepByID :one
SELECT * FROM recipe_steps WHERE id = $1;

-- name: ListRecipeSteps :many
SELECT * FROM recipe_steps WHERE recipe_id = $1 ORDER BY step_order;

-- name: UpdateRecipeStep :one
UPDATE recipe_steps SET description = $2, duration_min = $3, step_order = $4
WHERE id = $1 RETURNING *;

-- name: DeleteRecipeStep :exec
DELETE FROM recipe_steps WHERE id = $1;

-- name: DeleteRecipeStepsByRecipe :exec
DELETE FROM recipe_steps WHERE recipe_id = $1;

-- name: CreateStepIngredient :one
INSERT INTO step_ingredients (step_id, ingredient_id, unit_id, amount, notes)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListStepIngredients :many
SELECT
    si.id, si.step_id, si.ingredient_id, si.unit_id, si.amount, si.notes,
    i.name_en AS ingredient_name,
    COALESCE(i.name_de, '') AS ingredient_name_de,
    COALESCE(u.name_en, '') AS unit_name,
    COALESCE(u.name_de, '') AS unit_name_de,
    COALESCE(u.abbreviation_en, '') AS unit_abbr,
    COALESCE(u.abbreviation_de, '') AS unit_abbr_de
FROM step_ingredients si
JOIN ingredients i ON i.id = si.ingredient_id
LEFT JOIN units u ON u.id = si.unit_id
WHERE si.step_id = $1;

-- name: DeleteStepIngredient :exec
DELETE FROM step_ingredients WHERE id = $1;

-- name: DeleteStepIngredientsByStep :exec
DELETE FROM step_ingredients WHERE step_id = $1;

-- name: AddRecipeTag :exec
INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveRecipeTag :exec
DELETE FROM recipe_tags WHERE recipe_id = $1 AND tag_id = $2;

-- name: RemoveAllRecipeTags :exec
DELETE FROM recipe_tags WHERE recipe_id = $1;

-- name: ListRecipeTags :many
SELECT t.* FROM tags t
JOIN recipe_tags rt ON rt.tag_id = t.id
WHERE rt.recipe_id = $1;

-- name: CreateTag :one
INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING *;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = $1;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: CreateShareLink :one
INSERT INTO share_links (token, user_id, recipe_id, expires_at)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetShareLinkByToken :one
SELECT * FROM share_links WHERE token = $1;

-- name: ListUserShareLinks :many
SELECT * FROM share_links WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = $1 AND user_id = $2;
