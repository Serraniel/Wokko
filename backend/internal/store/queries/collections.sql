-- name: CreateBook :one
INSERT INTO recipe_books (owner_id, name, description) VALUES (?, ?, ?) RETURNING *;

-- name: GetBook :one
SELECT * FROM recipe_books WHERE id = ?;

-- name: UpdateBook :exec
UPDATE recipe_books SET name = ?, description = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = ?;

-- name: DeleteBook :exec
DELETE FROM recipe_books WHERE id = ?;

-- name: ListBooksForHousehold :many
SELECT b.* FROM recipe_books b JOIN users u ON u.id = b.owner_id
WHERE u.household_id = sqlc.arg(household_id) ORDER BY b.name COLLATE NOCASE, b.id;

-- name: AddBookEntry :one
INSERT INTO recipe_book_entries (book_id, recipe_id, added_by) VALUES (?, ?, ?)
ON CONFLICT (book_id, recipe_id) DO UPDATE SET book_id = excluded.book_id RETURNING *;

-- name: RemoveBookEntry :exec
DELETE FROM recipe_book_entries WHERE book_id = ? AND recipe_id = ?;

-- name: GetBookEntry :one
SELECT * FROM recipe_book_entries WHERE id = ?;

-- name: DeleteBookEntry :exec
DELETE FROM recipe_book_entries WHERE id = ?;

-- name: ListBookEntries :many
SELECT * FROM recipe_book_entries WHERE book_id = ? ORDER BY id;

-- name: ListBookEntriesForRecipe :many
SELECT * FROM recipe_book_entries WHERE recipe_id = ? ORDER BY id;

-- name: CreateCollectionLink :one
INSERT INTO collection_links (token, owner_id, expires_at) VALUES (?, ?, ?) RETURNING *;

-- name: GetCollectionLinkByToken :one
SELECT * FROM collection_links WHERE token = ?;

-- name: ListCollectionLinks :many
SELECT * FROM collection_links WHERE owner_id = ? ORDER BY id;

-- name: DeleteCollectionLink :execrows
DELETE FROM collection_links WHERE id = ? AND owner_id = ?;

-- name: ListCollectionRecipes :many
SELECT * FROM recipes
WHERE owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL AND visibility <> 'private'
  AND (CAST(sqlc.arg(q) AS TEXT) = '' OR search_text LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%')
ORDER BY name COLLATE NOCASE, id LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountCollectionRecipes :one
SELECT count(*) FROM recipes
WHERE owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL AND visibility <> 'private'
  AND (CAST(sqlc.arg(q) AS TEXT) = '' OR search_text LIKE '%' || CAST(sqlc.arg(q) AS TEXT) || '%');
