-- The starter catalog ships as data in the binary. These tables let a newer one reach existing
-- databases without overwriting anyone's work.

-- What this database last took from the shipped catalog, per entry, as a JSON object of fields.
-- entity_id is NULL once the entry was deleted or merged away here, so it never comes back.
CREATE TABLE catalog_entries (
    kind      TEXT NOT NULL CHECK (kind IN ('food', 'unit', 'tag')),
    key       TEXT NOT NULL,
    entity_id INTEGER,
    base      TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (kind, key)
) STRICT;
CREATE INDEX idx_catalog_entries_entity ON catalog_entries(kind, entity_id);

-- Changes the admin decides on: someone edited a value that the newer catalog also changed, or
-- the entry belongs to a person.
CREATE TABLE catalog_suggestions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    kind       TEXT NOT NULL,
    key        TEXT NOT NULL,
    entity_id  INTEGER NOT NULL,
    field      TEXT NOT NULL,
    ours       TEXT NOT NULL,
    theirs     TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE (kind, key, field)
) STRICT;

-- One row per catalog that was brought in, so the same one isn't processed twice.
CREATE TABLE catalog_syncs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    pack       TEXT NOT NULL,
    version    TEXT NOT NULL,
    added      INTEGER NOT NULL,
    updated    INTEGER NOT NULL,
    suggested  INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
