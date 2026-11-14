-- How much a recipe makes, such as 1.5 l of stock. Lets a recipe that uses this one as an
-- ingredient scale it: 450 ml of a stock that makes 1.5 l is 0.3 of the recipe.
ALTER TABLE recipes ADD COLUMN yield_amount REAL;
ALTER TABLE recipes ADD COLUMN yield_unit_id INTEGER REFERENCES units(id) ON DELETE SET NULL;
