-- name: CreateFood :one
INSERT INTO foods (default_unit_id, created_by) VALUES (?, ?) RETURNING *;

-- name: GetFood :one
SELECT * FROM foods WHERE id = ?;

-- name: SetFoodDefaultUnit :exec
UPDATE foods SET default_unit_id = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: UpsertFoodName :exec
INSERT INTO food_names (food_id, locale, name, plural, search_name) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (food_id, locale) DO UPDATE SET name = excluded.name, plural = excluded.plural, search_name = excluded.search_name;

-- name: DeleteFoodNames :exec
DELETE FROM food_names WHERE food_id = ?;

-- name: ListFoodNames :many
SELECT * FROM food_names WHERE food_id IN (sqlc.slice(ids));

-- name: SearchFoodIDs :many
SELECT f.id FROM foods f
WHERE CAST(sqlc.arg(q) AS TEXT) = ''
   OR EXISTS (SELECT 1 FROM food_names n WHERE n.food_id = f.id AND n.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
   OR EXISTS (SELECT 1 FROM food_aliases a WHERE a.food_id = f.id AND a.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
ORDER BY
  (SELECT count(*) FROM ingredients i WHERE i.food_id = f.id) DESC,
  f.id
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountFoods :one
SELECT count(*) FROM foods f
WHERE CAST(sqlc.arg(q) AS TEXT) = ''
   OR EXISTS (SELECT 1 FROM food_names n WHERE n.food_id = f.id AND n.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
   OR EXISTS (SELECT 1 FROM food_aliases a WHERE a.food_id = f.id AND a.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%');

-- name: ListFoodDefaults :many
SELECT id, default_unit_id FROM foods WHERE id IN (sqlc.slice(ids));

-- name: MostUsedUnitForFood :one
SELECT unit_id FROM ingredients WHERE food_id = ? AND unit_id IS NOT NULL
GROUP BY unit_id ORDER BY count(*) DESC LIMIT 1;

-- name: ListFoodAliases :many
SELECT * FROM food_aliases WHERE food_id IN (sqlc.slice(ids)) ORDER BY alias;

-- name: AddFoodAlias :exec
INSERT INTO food_aliases (food_id, alias, search_name) VALUES (?, ?, ?) ON CONFLICT DO NOTHING;

-- name: DeleteFoodAliases :exec
DELETE FROM food_aliases WHERE food_id = ?;

-- name: MoveIngredientsToFood :exec
UPDATE ingredients SET food_id = sqlc.arg(target) WHERE food_id = sqlc.arg(source);

-- name: DeleteFood :exec
DELETE FROM foods WHERE id = ?;

-- name: CountFoodUses :one
SELECT count(*) FROM ingredients WHERE food_id = ?;

-- name: CreateUnit :one
INSERT INTO units (created_by) VALUES (?) RETURNING *;

-- name: GetUnit :one
SELECT * FROM units WHERE id = ?;

-- name: UpsertUnitName :exec
INSERT INTO unit_names (unit_id, locale, name, plural, abbreviation, search_name) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (unit_id, locale) DO UPDATE SET name = excluded.name, plural = excluded.plural,
  abbreviation = excluded.abbreviation, search_name = excluded.search_name;

-- name: DeleteUnitNames :exec
DELETE FROM unit_names WHERE unit_id = ?;

-- name: ListUnitNames :many
SELECT * FROM unit_names WHERE unit_id IN (sqlc.slice(ids));

-- name: SearchUnitIDs :many
SELECT u.id FROM units u
WHERE CAST(sqlc.arg(q) AS TEXT) = ''
   OR EXISTS (SELECT 1 FROM unit_names n WHERE n.unit_id = u.id AND n.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
   OR EXISTS (SELECT 1 FROM unit_aliases a WHERE a.unit_id = u.id AND a.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
ORDER BY (SELECT count(*) FROM ingredients i WHERE i.unit_id = u.id) DESC, u.id
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountUnits :one
SELECT count(*) FROM units u
WHERE CAST(sqlc.arg(q) AS TEXT) = ''
   OR EXISTS (SELECT 1 FROM unit_names n WHERE n.unit_id = u.id AND n.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
   OR EXISTS (SELECT 1 FROM unit_aliases a WHERE a.unit_id = u.id AND a.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%');

-- name: ListUnitAliases :many
SELECT * FROM unit_aliases WHERE unit_id IN (sqlc.slice(ids)) ORDER BY alias;

-- name: AddUnitAlias :exec
INSERT INTO unit_aliases (unit_id, alias, search_name) VALUES (?, ?, ?) ON CONFLICT DO NOTHING;

-- name: DeleteUnitAliases :exec
DELETE FROM unit_aliases WHERE unit_id = ?;

-- name: MoveIngredientsToUnit :exec
UPDATE ingredients SET unit_id = sqlc.arg(target) WHERE unit_id = sqlc.arg(source);

-- name: MoveFoodDefaultsToUnit :exec
UPDATE foods SET default_unit_id = sqlc.arg(target) WHERE default_unit_id = sqlc.arg(source);

-- name: DeleteUnit :exec
DELETE FROM units WHERE id = ?;

-- name: CountUnitUses :one
SELECT count(*) FROM ingredients WHERE unit_id = ?;

-- name: CreateKeyword :one
INSERT INTO keywords DEFAULT VALUES RETURNING *;

-- name: GetKeyword :one
SELECT * FROM keywords WHERE id = ?;

-- name: UpsertKeywordName :exec
INSERT INTO keyword_names (keyword_id, locale, name, search_name) VALUES (?, ?, ?, ?)
ON CONFLICT (keyword_id, locale) DO UPDATE SET name = excluded.name, search_name = excluded.search_name;

-- name: ListKeywordNames :many
SELECT * FROM keyword_names WHERE keyword_id IN (sqlc.slice(ids));

-- name: FindKeywordBySearchName :one
SELECT keyword_id FROM keyword_names WHERE search_name = ? LIMIT 1;

-- name: ListKeywordsRows :many
SELECT * FROM keywords WHERE id IN (sqlc.slice(ids));

-- name: SearchKeywordIDs :many
SELECT k.id FROM keywords k
WHERE CAST(sqlc.arg(q) AS TEXT) = '' OR EXISTS (SELECT 1 FROM keyword_names n WHERE n.keyword_id = k.id AND n.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
ORDER BY (SELECT count(*) FROM recipe_keywords r WHERE r.keyword_id = k.id) DESC, k.id
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountKeywords :one
SELECT count(*) FROM keywords k
WHERE CAST(sqlc.arg(q) AS TEXT) = '' OR EXISTS (SELECT 1 FROM keyword_names n WHERE n.keyword_id = k.id AND n.search_name LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%');

-- name: FindFoodIDByName :one
SELECT food_id FROM food_names WHERE name = sqlc.arg(name) COLLATE NOCASE OR plural = sqlc.arg(name) COLLATE NOCASE LIMIT 1;

-- name: FindUnitIDByName :one
SELECT unit_id FROM unit_names WHERE name = sqlc.arg(name) COLLATE NOCASE OR plural = sqlc.arg(name) COLLATE NOCASE OR abbreviation = sqlc.arg(name) COLLATE NOCASE LIMIT 1;

-- name: ListAllUnitSpellings :many
SELECT name, plural, abbreviation FROM unit_names;

-- name: ListAllFoodSearchNames :many
SELECT food_id, search_name FROM food_names WHERE food_id IN (sqlc.slice(ids));
