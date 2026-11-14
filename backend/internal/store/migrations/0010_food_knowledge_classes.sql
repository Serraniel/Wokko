-- Kitchen knowledge classes someone set for a food, such as dairy.cream. They replace what
-- Wokko recognises from the food's names, for foods its word lists don't know or get wrong.
CREATE TABLE food_knowledge_classes (
    food_id INTEGER NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    class   TEXT NOT NULL,
    PRIMARY KEY (food_id, class)
) STRICT;
