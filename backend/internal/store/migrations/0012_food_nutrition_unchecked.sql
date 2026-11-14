-- 1 while a food's nutrition values were filled by an assistant and no person has saved them
-- since, so the app can mark them as unchecked.
ALTER TABLE foods ADD COLUMN nutrition_unchecked INTEGER NOT NULL DEFAULT 0;
