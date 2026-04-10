DELETE FROM tags;
DELETE FROM ingredient_unit_stats;
DELETE FROM ingredient_aliases;
DELETE FROM ingredients;
DELETE FROM unit_aliases;
DELETE FROM units;
DELETE FROM system_settings WHERE key = 'data_seeded';
