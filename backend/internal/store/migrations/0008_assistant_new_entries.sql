-- Whether an assistant may add foods and units to the catalog when it saves a recipe:
-- never, only after the person confirms them, or freely.
ALTER TABLE users ADD COLUMN assistant_new_entries TEXT NOT NULL DEFAULT 'ask' CHECK (assistant_new_entries IN ('never', 'ask', 'allow'));
