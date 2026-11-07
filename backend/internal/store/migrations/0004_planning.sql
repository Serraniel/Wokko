-- Meal plans, shopping and supermarkets belong to a household, and a person's own household
-- starts with just them. Removing a household removes its planning data, never its recipes.
CREATE TABLE meal_types (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    household_id INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    position     INTEGER NOT NULL DEFAULT 0,
    time         TEXT NOT NULL DEFAULT '',
    color        TEXT NOT NULL DEFAULT '',
    created_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_meal_types_household ON meal_types(household_id, position);

-- recipe_name is a snapshot, so a plan still says what it was after a recipe is purged.
CREATE TABLE meal_plans (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    household_id INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    created_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
    recipe_id    INTEGER REFERENCES recipes(id) ON DELETE SET NULL,
    recipe_name  TEXT NOT NULL DEFAULT '',
    meal_type_id INTEGER NOT NULL REFERENCES meal_types(id) ON DELETE CASCADE,
    title        TEXT NOT NULL DEFAULT '',
    note         TEXT NOT NULL DEFAULT '',
    servings     REAL NOT NULL DEFAULT 1,
    from_date    TEXT NOT NULL,
    to_date      TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_meal_plans_dates ON meal_plans(household_id, from_date, to_date);
CREATE INDEX idx_meal_plans_recipe ON meal_plans(recipe_id);

CREATE TABLE supermarket_categories (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    household_id INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_categories_household ON supermarket_categories(household_id);

CREATE TABLE supermarkets (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    household_id INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_supermarkets_household ON supermarkets(household_id);

CREATE TABLE supermarket_category_order (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    supermarket_id INTEGER NOT NULL REFERENCES supermarkets(id) ON DELETE CASCADE,
    category_id    INTEGER NOT NULL REFERENCES supermarket_categories(id) ON DELETE CASCADE,
    position       INTEGER NOT NULL DEFAULT 0,
    UNIQUE (supermarket_id, category_id)
) STRICT;

-- The food catalog is shared by the whole instance, while how a household sorts its shopping is its own.
CREATE TABLE food_categories (
    household_id INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    food_id      INTEGER NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    category_id  INTEGER NOT NULL REFERENCES supermarket_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (household_id, food_id)
) STRICT;

CREATE TABLE shopping_recipes (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    household_id INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    recipe_id    INTEGER REFERENCES recipes(id) ON DELETE SET NULL,
    meal_plan_id INTEGER REFERENCES meal_plans(id) ON DELETE SET NULL,
    name         TEXT NOT NULL,
    servings     REAL NOT NULL DEFAULT 1,
    created_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_shopping_recipes_household ON shopping_recipes(household_id);

CREATE TABLE shopping_items (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    household_id       INTEGER NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    shopping_recipe_id INTEGER REFERENCES shopping_recipes(id) ON DELETE SET NULL,
    food_id            INTEGER NOT NULL REFERENCES foods(id) ON DELETE RESTRICT,
    unit_id            INTEGER REFERENCES units(id) ON DELETE RESTRICT,
    amount             REAL,
    note               TEXT NOT NULL DEFAULT '',
    position           INTEGER NOT NULL DEFAULT 0,
    checked_at         TEXT,
    created_by         INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
) STRICT;
CREATE INDEX idx_shopping_items_household ON shopping_items(household_id, position);
CREATE INDEX idx_shopping_items_food ON shopping_items(food_id);
CREATE INDEX idx_shopping_items_unit ON shopping_items(unit_id);

-- Nutrition. Values are per 100 g of a food. units.grams says what one unit weighs, with
-- volumes taken at the density of water, and foods.piece_grams what one piece weighs.
ALTER TABLE foods ADD COLUMN piece_grams REAL;
ALTER TABLE units ADD COLUMN grams REAL;

CREATE TABLE nutrients (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    unit     TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE TABLE nutrient_names (
    nutrient_id INTEGER NOT NULL REFERENCES nutrients(id) ON DELETE CASCADE,
    locale      TEXT NOT NULL,
    name        TEXT NOT NULL,
    PRIMARY KEY (nutrient_id, locale)
) STRICT;
CREATE TABLE food_nutrients (
    food_id     INTEGER NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    nutrient_id INTEGER NOT NULL REFERENCES nutrients(id) ON DELETE CASCADE,
    per_100g    REAL NOT NULL CHECK (per_100g >= 0),
    PRIMARY KEY (food_id, nutrient_id)
) STRICT;

INSERT INTO nutrients (id, unit, position) VALUES (1,'kcal',1),(2,'g',2),(3,'g',3),(4,'g',4),(5,'g',5),(6,'g',6),(7,'g',7),(8,'g',8);
INSERT INTO nutrient_names (nutrient_id, locale, name) VALUES
 (1,'en','Energy'),(1,'de','Energie'),(2,'en','Fat'),(2,'de','Fett'),(3,'en','Saturated fat'),(3,'de','Gesättigte Fettsäuren'),
 (4,'en','Carbohydrates'),(4,'de','Kohlenhydrate'),(5,'en','Sugar'),(5,'de','Zucker'),(6,'en','Fiber'),(6,'de','Ballaststoffe'),
 (7,'en','Protein'),(7,'de','Eiweiß'),(8,'en','Salt'),(8,'de','Salz');

UPDATE units SET grams = 1 WHERE id = 1;
UPDATE units SET grams = 1000 WHERE id = 2;
UPDATE units SET grams = 0.001 WHERE id = 3;
UPDATE units SET grams = 1 WHERE id = 4;
UPDATE units SET grams = 1000 WHERE id = 5;
UPDATE units SET grams = 10 WHERE id = 6;
UPDATE units SET grams = 5 WHERE id = 7;
UPDATE units SET grams = 15 WHERE id = 8;
UPDATE units SET grams = 240 WHERE id = 9;
UPDATE units SET grams = 30 WHERE id = 10;
UPDATE units SET grams = 28.35 WHERE id = 11;
UPDATE units SET grams = 453.6 WHERE id = 12;
