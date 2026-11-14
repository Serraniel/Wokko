-- An optional email address per account, for mail when the server sends any. It's only used
-- once confirmed, and only shown to the person and admins.
ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX idx_users_verified_email ON users(lower(email)) WHERE email_verified = 1 AND email != '';

-- Links that confirm an address. Only the hash of the token is stored.
CREATE TABLE email_confirmations (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    expires_at TEXT NOT NULL
) STRICT;
