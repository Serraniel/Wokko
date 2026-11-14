-- Where a food's nutrition values came from, such as a dataset entry or "package label".
-- Shown next to the values, so people know how far to trust them.
ALTER TABLE foods ADD COLUMN nutrition_source TEXT NOT NULL DEFAULT '';
