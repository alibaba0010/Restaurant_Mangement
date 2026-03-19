-- Up: Add address_id to users and restaurants to reference the addresses table
-- This allows a direct link to the primary/active address while maintaining 
-- the one-to-many relationship for history/multiple locations.

-- 1. Add address_id columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS address_id UUID REFERENCES addresses(id) ON DELETE SET NULL;
ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS address_id UUID REFERENCES addresses(id) ON DELETE SET NULL;

-- 2. Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_address_id ON users(address_id);
CREATE INDEX IF NOT EXISTS idx_restaurants_address_id ON restaurants(address_id);

-- 3. Migrate existing default addresses to the new columns
-- For users:
UPDATE users u
SET address_id = a.id
FROM addresses a
WHERE a.user_id::text = u.id::text AND a.is_default = true;

-- For restaurants:
UPDATE restaurants r
SET address_id = a.id
FROM addresses a
WHERE a.restaurant_id = r.id AND a.is_default = true;

-- 4. Clean up legacy/redundant columns from restaurants
-- The 'address', 'latitude', and 'longitude' fields are now handled by the addresses table
ALTER TABLE restaurants DROP COLUMN IF EXISTS address;
ALTER TABLE restaurants DROP COLUMN IF EXISTS latitude;
ALTER TABLE restaurants DROP COLUMN IF EXISTS longitude;
