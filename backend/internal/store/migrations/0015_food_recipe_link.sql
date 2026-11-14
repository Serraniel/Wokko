-- A food that can be cooked from a recipe, such as chicken stock from its own recipe. Recipes
-- that use the food then link to it, for people who may see that recipe.
ALTER TABLE foods ADD COLUMN recipe_id INTEGER REFERENCES recipes(id) ON DELETE SET NULL;
CREATE INDEX idx_foods_recipe ON foods(recipe_id);
