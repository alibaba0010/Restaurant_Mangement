-- Enable pg_trgm extension for ILIKE optimization (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Indexes for search optimization (ILIKE queries)

-- Restaurants
CREATE INDEX IF NOT EXISTS idx_restaurants_name_trgm ON restaurants USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_restaurants_description_trgm ON restaurants USING gin (description gin_trgm_ops);

-- Users
CREATE INDEX IF NOT EXISTS idx_users_name_trgm ON users USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_email_trgm ON users USING gin (email gin_trgm_ops);

-- Menu
CREATE INDEX IF NOT EXISTS idx_menu_name_trgm ON menu USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_menu_description_trgm ON menu USING gin (description gin_trgm_ops);


-- Indexes for cursor pagination (composite indexes)
-- Support (sort_col, id) > (val, id) queries

-- Restaurants
CREATE INDEX IF NOT EXISTS idx_restaurants_created_at_id ON restaurants (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_restaurants_rating_id ON restaurants (rating DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_restaurants_capacity_id ON restaurants (capacity DESC, id DESC);

-- Menu
CREATE INDEX IF NOT EXISTS idx_menus_created_at_id ON menu (created_at DESC, id DESC);
-- Add price if sorted by price often, but created_at is default.

