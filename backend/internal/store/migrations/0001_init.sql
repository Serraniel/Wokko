-- Wokko schema. All tables are STRICT, all primary keys are stable integers that are
-- never reused (AUTOINCREMENT), and timestamps are RFC 3339 UTC text.
-- Content is never removed by a cascade from users or the shared catalog.

CREATE TABLE households (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;

CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    handle        TEXT NOT NULL UNIQUE COLLATE NOCASE,
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin','user')),
    locale        TEXT NOT NULL DEFAULT 'en',
    household_id  INTEGER NOT NULL REFERENCES households(id) ON DELETE RESTRICT,
    disabled_at   TEXT,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_users_household ON users(household_id);

CREATE TABLE sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_sessions_user ON sessions(user_id);

-- kind 'registration': lets a new user sign up. household_id set: also joins that household
-- (and lets an existing user join it).
CREATE TABLE invites (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash   TEXT NOT NULL UNIQUE,
    created_by   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    household_id INTEGER REFERENCES households(id) ON DELETE CASCADE,
    max_uses     INTEGER NOT NULL DEFAULT 1,
    use_count    INTEGER NOT NULL DEFAULT 0,
    expires_at   TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;

-- Admin-issued links, so a forgotten password never needs email.
CREATE TABLE password_resets (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
) STRICT;

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;
INSERT INTO settings (key, value) VALUES ('open_registration','false'), ('allow_public_recipes','true');

CREATE TABLE friendships (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    requester_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    accepted_at  TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id)
) STRICT;
CREATE INDEX idx_friend_addressee ON friendships(addressee_id);

-- Shared catalog. Names live in per-locale rows, so a new language needs no schema change.
CREATE TABLE units (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE TABLE unit_names (
    unit_id      INTEGER NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    locale       TEXT NOT NULL,
    name         TEXT NOT NULL,
    plural       TEXT,
    abbreviation TEXT,
    search_name  TEXT NOT NULL,
    PRIMARY KEY (unit_id, locale)
) STRICT;
CREATE INDEX idx_unit_names_search ON unit_names(search_name);

CREATE TABLE foods (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    default_unit_id INTEGER REFERENCES units(id) ON DELETE SET NULL,
    created_by      INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE TABLE food_names (
    food_id     INTEGER NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    locale      TEXT NOT NULL,
    name        TEXT NOT NULL,
    plural      TEXT,
    search_name TEXT NOT NULL,
    PRIMARY KEY (food_id, locale)
) STRICT;
CREATE INDEX idx_food_names_search ON food_names(search_name);
-- Alternative spellings. Merging foods keeps the merged names here so search still finds them.
CREATE TABLE food_aliases (
    food_id     INTEGER NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    alias       TEXT NOT NULL,
    search_name TEXT NOT NULL,
    PRIMARY KEY (food_id, alias)
) STRICT;
CREATE TABLE unit_aliases (
    unit_id     INTEGER NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    alias       TEXT NOT NULL,
    search_name TEXT NOT NULL,
    PRIMARY KEY (unit_id, alias)
) STRICT;

CREATE TABLE keywords (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE TABLE keyword_names (
    keyword_id  INTEGER NOT NULL REFERENCES keywords(id) ON DELETE CASCADE,
    locale      TEXT NOT NULL,
    name        TEXT NOT NULL,
    search_name TEXT NOT NULL,
    PRIMARY KEY (keyword_id, locale)
) STRICT;
CREATE INDEX idx_keyword_names_search ON keyword_names(search_name);

-- owner_id is RESTRICT: deleting a user requires an explicit choice about their recipes.
CREATE TABLE recipes (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    work_minutes  INTEGER NOT NULL DEFAULT 0,
    wait_minutes  INTEGER NOT NULL DEFAULT 0,
    servings      INTEGER NOT NULL DEFAULT 1,
    servings_text TEXT NOT NULL DEFAULT '',
    source_url    TEXT NOT NULL DEFAULT '',
    visibility    TEXT NOT NULL DEFAULT 'household' CHECK (visibility IN ('private','household','public')),
    photo         TEXT,
    version       INTEGER NOT NULL DEFAULT 1,
    search_text   TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    deleted_at    TEXT
) STRICT;
CREATE INDEX idx_recipes_owner ON recipes(owner_id);
CREATE INDEX idx_recipes_visibility ON recipes(visibility) WHERE deleted_at IS NULL;
CREATE INDEX idx_recipes_photo ON recipes(photo);

CREATE TABLE recipe_keywords (
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    keyword_id INTEGER NOT NULL REFERENCES keywords(id) ON DELETE CASCADE,
    PRIMARY KEY (recipe_id, keyword_id)
) STRICT;
CREATE INDEX idx_recipe_keywords_kw ON recipe_keywords(keyword_id);

CREATE TABLE steps (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id   INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    instruction TEXT NOT NULL DEFAULT '',
    minutes     INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE INDEX idx_steps_recipe ON steps(recipe_id, position);

-- food_id and unit_id are RESTRICT: catalog entries can't vanish from under a recipe.
CREATE TABLE ingredients (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    step_id  INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    food_id  INTEGER NOT NULL REFERENCES foods(id) ON DELETE RESTRICT,
    unit_id  INTEGER REFERENCES units(id) ON DELETE RESTRICT,
    amount   REAL,
    note     TEXT NOT NULL DEFAULT ''
) STRICT;
CREATE INDEX idx_ingredients_step ON ingredients(step_id, position);
CREATE INDEX idx_ingredients_food ON ingredients(food_id, unit_id);
CREATE INDEX idx_ingredients_unit ON ingredients(unit_id);

-- Full JSON snapshot taken before every overwrite.
CREATE TABLE recipe_versions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    version    INTEGER NOT NULL,
    document   TEXT NOT NULL,
    saved_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE (recipe_id, version)
) STRICT;

CREATE TABLE recipe_shares (
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    PRIMARY KEY (recipe_id, user_id)
) STRICT;
CREATE INDEX idx_recipe_shares_user ON recipe_shares(user_id);

CREATE TABLE share_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT NOT NULL UNIQUE,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_share_links_recipe ON share_links(recipe_id);

CREATE TABLE cook_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    servings   INTEGER,
    rating     INTEGER CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_cook_log_recipe ON cook_log(recipe_id, user_id);

CREATE TABLE backup_runs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at  TEXT NOT NULL,
    finished_at TEXT,
    status      TEXT NOT NULL CHECK (status IN ('running','ok','failed')),
    file        TEXT,
    size_bytes  INTEGER,
    error       TEXT
) STRICT;
