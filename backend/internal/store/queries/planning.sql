-- name: CreateMealType :one
INSERT INTO meal_types (household_id, name, position, time, color, created_by) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetMealType :one
SELECT * FROM meal_types WHERE id = ?;

-- name: ListMealTypes :many
SELECT * FROM meal_types WHERE household_id = ? ORDER BY position, id;

-- name: UpdateMealType :exec
UPDATE meal_types SET name = ?, position = ?, time = ?, color = ? WHERE id = ?;

-- name: DeleteMealType :exec
DELETE FROM meal_types WHERE id = ?;

-- name: CountMealTypeUses :one
SELECT count(*) FROM meal_plans WHERE meal_type_id = ?;

-- name: CreateMealPlan :one
INSERT INTO meal_plans (household_id, created_by, recipe_id, recipe_name, meal_type_id, title, note, servings, from_date, to_date)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetMealPlan :one
SELECT * FROM meal_plans WHERE id = ?;

-- name: UpdateMealPlan :exec
UPDATE meal_plans SET recipe_id = ?, recipe_name = ?, meal_type_id = ?, title = ?, note = ?, servings = ?, from_date = ?, to_date = ?,
  updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: DeleteMealPlan :exec
DELETE FROM meal_plans WHERE id = ?;

-- name: ListMealPlans :many
SELECT * FROM meal_plans WHERE household_id = sqlc.arg(household_id)
  AND (CAST(sqlc.arg(from_date) AS TEXT) = '' OR to_date >= CAST(sqlc.arg(from_date) AS TEXT))
  AND (CAST(sqlc.arg(to_date) AS TEXT) = '' OR from_date <= CAST(sqlc.arg(to_date) AS TEXT))
  AND (CAST(sqlc.arg(meal_type_id) AS INTEGER) = 0 OR meal_type_id = CAST(sqlc.arg(meal_type_id) AS INTEGER))
ORDER BY from_date, id;

-- name: CreateCategory :one
INSERT INTO supermarket_categories (household_id, name, description) VALUES (?, ?, ?) RETURNING *;

-- name: GetCategory :one
SELECT * FROM supermarket_categories WHERE id = ?;

-- name: ListCategories :many
SELECT * FROM supermarket_categories WHERE household_id = ? ORDER BY name COLLATE NOCASE, id;

-- name: UpdateCategory :exec
UPDATE supermarket_categories SET name = ?, description = ? WHERE id = ?;

-- name: DeleteCategory :exec
DELETE FROM supermarket_categories WHERE id = ?;

-- name: CreateSupermarket :one
INSERT INTO supermarkets (household_id, name, description) VALUES (?, ?, ?) RETURNING *;

-- name: GetSupermarket :one
SELECT * FROM supermarkets WHERE id = ?;

-- name: ListSupermarkets :many
SELECT * FROM supermarkets WHERE household_id = ? ORDER BY name COLLATE NOCASE, id;

-- name: UpdateSupermarket :exec
UPDATE supermarkets SET name = ?, description = ? WHERE id = ?;

-- name: DeleteSupermarket :exec
DELETE FROM supermarkets WHERE id = ?;

-- name: ClearSupermarketOrder :exec
DELETE FROM supermarket_category_order WHERE supermarket_id = ?;

-- name: AddSupermarketOrder :one
INSERT INTO supermarket_category_order (supermarket_id, category_id, position) VALUES (?, ?, ?)
ON CONFLICT (supermarket_id, category_id) DO UPDATE SET position = excluded.position RETURNING *;

-- name: GetSupermarketOrderEntry :one
SELECT * FROM supermarket_category_order WHERE id = ?;

-- name: UpdateSupermarketOrderEntry :exec
UPDATE supermarket_category_order SET position = ? WHERE id = ?;

-- name: DeleteSupermarketOrderEntry :exec
DELETE FROM supermarket_category_order WHERE id = ?;

-- name: ListSupermarketOrder :many
SELECT * FROM supermarket_category_order WHERE supermarket_id = ? ORDER BY position, id;

-- name: SetFoodCategory :exec
INSERT INTO food_categories (household_id, food_id, category_id) VALUES (?, ?, ?)
ON CONFLICT (household_id, food_id) DO UPDATE SET category_id = excluded.category_id;

-- name: ClearFoodCategory :exec
DELETE FROM food_categories WHERE household_id = ? AND food_id = ?;

-- name: ListFoodCategories :many
SELECT * FROM food_categories WHERE household_id = ?;

-- name: MoveFoodCategories :exec
UPDATE OR IGNORE food_categories SET food_id = sqlc.arg(target) WHERE food_id = sqlc.arg(source);

-- name: CreateShoppingRecipe :one
INSERT INTO shopping_recipes (household_id, recipe_id, meal_plan_id, name, servings, created_by) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetShoppingRecipe :one
SELECT * FROM shopping_recipes WHERE id = ?;

-- name: ListShoppingRecipes :many
SELECT * FROM shopping_recipes WHERE household_id = ? ORDER BY id;

-- name: DeleteShoppingRecipe :exec
DELETE FROM shopping_recipes WHERE id = ?;

-- name: UpdateShoppingRecipeServings :exec
UPDATE shopping_recipes SET servings = ? WHERE id = ?;

-- name: ListUnusedShoppingRecipes :many
SELECT r.id FROM shopping_recipes r LEFT JOIN shopping_items i ON i.shopping_recipe_id = r.id
WHERE r.household_id = ? AND i.id IS NULL;

-- name: CreateShoppingItem :one
INSERT INTO shopping_items (household_id, shopping_recipe_id, food_id, unit_id, amount, note, position, created_by, checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetShoppingItem :one
SELECT * FROM shopping_items WHERE id = ?;

-- name: ListShoppingItems :many
SELECT * FROM shopping_items WHERE household_id = ? ORDER BY position, id;

-- name: NextShoppingPosition :one
SELECT CAST(COALESCE(MAX(position), 0) + 1 AS INTEGER) FROM shopping_items WHERE household_id = ?;

-- name: UpdateShoppingItem :exec
UPDATE shopping_items SET food_id = ?, unit_id = ?, amount = ?, note = ?, position = ?, checked_at = ?,
  updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: SetShoppingItemChecked :exec
UPDATE shopping_items SET checked_at = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ? AND household_id = ?;

-- name: DeleteShoppingItem :exec
DELETE FROM shopping_items WHERE id = ?;

-- name: DeleteCheckedShoppingItems :exec
DELETE FROM shopping_items WHERE household_id = ? AND checked_at IS NOT NULL;

-- name: DeleteShoppingItemsOfRecipe :exec
DELETE FROM shopping_items WHERE shopping_recipe_id = ?;

-- name: MoveShoppingItemsToFood :exec
UPDATE shopping_items SET food_id = sqlc.arg(target) WHERE food_id = sqlc.arg(source);

-- name: MoveShoppingItemsToUnit :exec
UPDATE shopping_items SET unit_id = sqlc.arg(target) WHERE unit_id = sqlc.arg(source);

-- name: CountFoodShoppingUses :one
SELECT count(*) FROM shopping_items WHERE food_id = ?;

-- name: CountUnitShoppingUses :one
SELECT count(*) FROM shopping_items WHERE unit_id = ?;

-- name: ListNutrients :many
SELECT n.id, n.unit, n.position, CAST(COALESCE(
  (SELECT x.name FROM nutrient_names x WHERE x.nutrient_id = n.id AND x.locale = sqlc.arg(locale)),
  (SELECT x.name FROM nutrient_names x WHERE x.nutrient_id = n.id AND x.locale = 'en')) AS TEXT) AS name
FROM nutrients n ORDER BY n.position, n.id;

-- name: ListFoodNutrients :many
SELECT * FROM food_nutrients WHERE food_id IN (sqlc.slice(ids));

-- name: ClearFoodNutrients :exec
DELETE FROM food_nutrients WHERE food_id = ?;

-- name: SetFoodNutrient :exec
INSERT INTO food_nutrients (food_id, nutrient_id, per_100g) VALUES (?, ?, ?)
ON CONFLICT (food_id, nutrient_id) DO UPDATE SET per_100g = excluded.per_100g;

-- name: MoveFoodNutrients :exec
UPDATE OR IGNORE food_nutrients SET food_id = sqlc.arg(target) WHERE food_id = sqlc.arg(source);

-- name: SetFoodPieceGrams :exec
UPDATE foods SET piece_grams = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: ListFoodPieceGrams :many
SELECT id, piece_grams FROM foods WHERE id IN (sqlc.slice(ids));

-- name: ListUnitGrams :many
SELECT id, grams FROM units WHERE id IN (sqlc.slice(ids));
