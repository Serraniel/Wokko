-- 1 for an account an admin created that hasn't set its password yet.
ALTER TABLE users ADD COLUMN needs_setup INTEGER NOT NULL DEFAULT 0;
-- A note on an invite link, such as who it's for.
ALTER TABLE invites ADD COLUMN note TEXT NOT NULL DEFAULT '';
