-- name: CreateRecipe :one
INSERT INTO recipes (owner_id, name, description, work_minutes, wait_minutes, servings, servings_text, source_url, visibility, search_text)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes WHERE id = ?;

-- name: UpdateRecipe :execrows
UPDATE recipes SET name = ?, description = ?, work_minutes = ?, wait_minutes = ?, servings = ?, servings_text = ?,
  source_url = ?, visibility = ?, search_text = ?, version = version + 1,
  updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = sqlc.arg(id) AND version = sqlc.arg(base_version) AND deleted_at IS NULL;

-- name: SetRecipePhoto :exec
UPDATE recipes SET photo = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: GetRecipeByPhoto :one
SELECT * FROM recipes WHERE photo = ? LIMIT 1;

-- name: ListAllPhotos :many
SELECT photo FROM recipes WHERE photo IS NOT NULL;

-- name: SoftDeleteRecipe :exec
UPDATE recipes SET deleted_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ? AND deleted_at IS NULL;

-- name: RestoreRecipe :exec
UPDATE recipes SET deleted_at = NULL WHERE id = ?;

-- name: PurgeRecipe :exec
DELETE FROM recipes WHERE id = ? AND deleted_at IS NOT NULL;

-- name: ListTrash :many
SELECT * FROM recipes WHERE owner_id = ? AND deleted_at IS NOT NULL ORDER BY deleted_at DESC;

-- name: ListExpiredTrash :many
SELECT * FROM recipes WHERE deleted_at IS NOT NULL AND deleted_at < ?;

-- name: ListVisibleRecipeKeys :many
SELECT r.id, r.name, r.created_at, r.updated_at FROM recipes r
WHERE r.deleted_at IS NULL
  AND (r.owner_id = sqlc.arg(viewer_id)
       OR (r.visibility IN ('household','public') AND r.owner_id IN (SELECT u.id FROM users u WHERE u.household_id = sqlc.arg(household_id)))
       OR r.id IN (SELECT s.recipe_id FROM recipe_shares s WHERE s.user_id = sqlc.arg(viewer_id)))
  AND (CAST(sqlc.arg(q) AS TEXT) = '' OR r.search_text LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
  AND (CAST(sqlc.arg(keyword_id) AS INTEGER) = 0 OR r.id IN (SELECT rk.recipe_id FROM recipe_keywords rk WHERE rk.keyword_id = CAST(sqlc.arg(keyword_id) AS INTEGER)))
  AND (CAST(sqlc.arg(owner_filter) AS INTEGER) = 0 OR r.owner_id = CAST(sqlc.arg(owner_filter) AS INTEGER));

-- name: ListRecipesByIDs :many
SELECT * FROM recipes WHERE id IN (sqlc.slice(ids));

-- name: ListPublicRecipes :many
SELECT * FROM recipes WHERE visibility = 'public' AND deleted_at IS NULL
  AND (CAST(sqlc.arg(q) AS TEXT) = '' OR search_text LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
ORDER BY updated_at DESC, id DESC LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountPublicRecipes :one
SELECT count(*) FROM recipes WHERE visibility = 'public' AND deleted_at IS NULL
  AND (CAST(sqlc.arg(q) AS TEXT) = '' OR search_text LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%');

-- name: ListRecipesByOwner :many
SELECT * FROM recipes WHERE owner_id = ? ORDER BY id;

-- name: CountRecipesByOwner :one
SELECT count(*) FROM recipes WHERE owner_id = ?;

-- name: ReassignRecipes :exec
UPDATE recipes SET owner_id = sqlc.arg(target) WHERE owner_id = sqlc.arg(source);

-- name: DeleteRecipesByOwner :exec
DELETE FROM recipes WHERE owner_id = ?;

-- name: IsSharedWith :one
SELECT count(*) FROM recipe_shares WHERE recipe_id = ? AND user_id = ?;

-- name: DeleteSteps :exec
DELETE FROM steps WHERE recipe_id = ?;

-- name: CreateStep :one
INSERT INTO steps (recipe_id, position, name, instruction, minutes) VALUES (?, ?, ?, ?, ?) RETURNING id;

-- name: CreateIngredient :exec
INSERT INTO ingredients (step_id, position, food_id, unit_id, amount, note) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListSteps :many
SELECT * FROM steps WHERE recipe_id = ? ORDER BY position, id;

-- name: GetStep :one
SELECT * FROM steps WHERE id = ?;

-- name: ListIngredientsForRecipe :many
SELECT i.* FROM ingredients i JOIN steps s ON s.id = i.step_id WHERE s.recipe_id = ? ORDER BY s.position, i.position, i.id;

-- name: DeleteRecipeKeywords :exec
DELETE FROM recipe_keywords WHERE recipe_id = ?;

-- name: AddRecipeKeyword :exec
INSERT INTO recipe_keywords (recipe_id, keyword_id) VALUES (?, ?) ON CONFLICT DO NOTHING;

-- name: ListRecipeKeywordsFor :many
SELECT * FROM recipe_keywords WHERE recipe_id IN (sqlc.slice(ids));

-- name: CreateRecipeVersion :exec
INSERT INTO recipe_versions (recipe_id, version, document, saved_by) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING;

-- name: PruneRecipeVersions :exec
DELETE FROM recipe_versions WHERE id IN
  (SELECT v.id FROM recipe_versions v WHERE v.recipe_id = ? ORDER BY v.id DESC LIMIT -1 OFFSET 50);

-- name: ListRecipeVersions :many
SELECT id, recipe_id, version, saved_by, created_at FROM recipe_versions WHERE recipe_id = ? ORDER BY id DESC;

-- name: GetRecipeVersion :one
SELECT * FROM recipe_versions WHERE recipe_id = ? AND id = ?;

-- name: AddRecipeShare :exec
INSERT INTO recipe_shares (recipe_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING;

-- name: RemoveRecipeShare :exec
DELETE FROM recipe_shares WHERE recipe_id = ? AND user_id = ?;

-- name: ListRecipeShareUsers :many
SELECT u.* FROM recipe_shares s JOIN users u ON u.id = s.user_id WHERE s.recipe_id = ? ORDER BY u.handle;

-- name: CreateShareLink :one
INSERT INTO share_links (token, recipe_id, created_by, expires_at) VALUES (?, ?, ?, ?) RETURNING *;

-- name: GetShareLinkByToken :one
SELECT * FROM share_links WHERE token = ?;

-- name: ListShareLinks :many
SELECT * FROM share_links WHERE recipe_id = ? ORDER BY id DESC;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = ? AND recipe_id = ?;

-- name: CountValidShareLink :one
SELECT count(*) FROM share_links l JOIN recipes r ON r.id = l.recipe_id
WHERE l.token = ? AND l.recipe_id = ? AND r.deleted_at IS NULL AND (l.expires_at IS NULL OR l.expires_at > ?);

-- name: CreateCookLog :one
INSERT INTO cook_log (recipe_id, user_id, servings, rating, comment) VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: ListCookLog :many
SELECT * FROM cook_log WHERE recipe_id = ? AND user_id = ? ORDER BY id DESC;

-- name: RecipeStats :many
SELECT recipe_id, CAST(AVG(rating) AS REAL) AS rating, CAST(MAX(created_at) AS TEXT) AS last_cooked, count(*) AS times_cooked
FROM cook_log WHERE user_id = sqlc.arg(user_id) AND recipe_id IN (sqlc.slice(ids)) GROUP BY recipe_id;

-- name: CreatePasswordReset :exec
INSERT INTO password_resets (token_hash, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetPasswordReset :one
SELECT * FROM password_resets WHERE token_hash = ? AND expires_at > ?;

-- name: DeletePasswordResets :exec
DELETE FROM password_resets WHERE user_id = ?;

-- name: CountAllRecipes :one
SELECT count(*) FROM recipes WHERE deleted_at IS NULL;

-- name: CountAllFoods :one
SELECT count(*) FROM foods;

-- name: CountAllUnits :one
SELECT count(*) FROM units;

-- name: StartBackupRun :one
INSERT INTO backup_runs (started_at, status) VALUES (?, 'running') RETURNING id;

-- name: FinishBackupRun :exec
UPDATE backup_runs SET finished_at = ?, status = ?, file = ?, size_bytes = ?, error = ? WHERE id = ?;

-- name: LastBackupRun :one
SELECT * FROM backup_runs ORDER BY id DESC LIMIT 1;

-- name: LastSuccessfulBackupRun :one
SELECT * FROM backup_runs WHERE status = 'ok' ORDER BY id DESC LIMIT 1;

-- name: ListBackupRuns :many
SELECT * FROM backup_runs ORDER BY id DESC LIMIT ?;

-- name: UpdateStep :exec
UPDATE steps SET position = ?, name = ?, instruction = ?, minutes = ? WHERE id = ?;

-- name: DeleteStep :exec
DELETE FROM steps WHERE id = ?;

-- name: DeleteStepIngredients :exec
DELETE FROM ingredients WHERE step_id = ?;

-- name: SetRecipeSearchText :exec
UPDATE recipes SET search_text = ? WHERE id = ?;

-- name: ListRecipeIDsUsingFood :many
SELECT DISTINCT s.recipe_id FROM ingredients i JOIN steps s ON s.id = i.step_id WHERE i.food_id = ?;

-- name: ListStepTexts :many
SELECT name, instruction FROM steps WHERE recipe_id = ?;

-- name: ListIngredientFacts :many
SELECT i.food_id, i.note FROM ingredients i JOIN steps s ON s.id = i.step_id WHERE s.recipe_id = ?;

-- name: ListFoodAliasSearchNames :many
SELECT search_name FROM food_aliases WHERE food_id IN (sqlc.slice(ids));

-- name: ListKeywordSearchNamesForRecipe :many
SELECT n.search_name FROM keyword_names n JOIN recipe_keywords k ON k.keyword_id = n.keyword_id WHERE k.recipe_id = ?;
