-- Recipe books are visible to the owner's household. Entries hold only a reference, so
-- deleting a book never touches a recipe.
CREATE TABLE recipe_books (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_recipe_books_owner ON recipe_books(owner_id);

CREATE TABLE recipe_book_entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    INTEGER NOT NULL REFERENCES recipe_books(id) ON DELETE CASCADE,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    added_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE (book_id, recipe_id)
) STRICT;
CREATE INDEX idx_book_entries_recipe ON recipe_book_entries(recipe_id);

-- A link to everything its owner hasn't kept private.
CREATE TABLE collection_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT NOT NULL UNIQUE,
    owner_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_collection_links_owner ON collection_links(owner_id);
