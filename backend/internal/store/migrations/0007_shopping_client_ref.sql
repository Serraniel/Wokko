-- An ID the app picks for an item added while offline. If the server saved an item but the
-- answer got lost, the app sends it again, and this is how the repeat is recognised.
ALTER TABLE shopping_items ADD COLUMN client_ref TEXT;
CREATE UNIQUE INDEX idx_shopping_items_client_ref ON shopping_items(household_id, client_ref) WHERE client_ref IS NOT NULL;
