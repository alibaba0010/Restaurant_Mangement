-- Down: Revert address_id changes and restore redundant columns
-- WARNING: Some data in 'address' column might be lost or need to be reconstructed from default address

-- 1. Restore redundant columns to restaurants
ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS latitude DECIMAL(10, 8);
ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS longitude DECIMAL(11, 8);

-- 2. Attempt to restore data from default address
UPDATE restaurants r
SET address = a.raw_address,
    latitude = a.latitude,
    longitude = a.longitude
FROM addresses a
WHERE r.address_id = a.id;

-- 3. Drop indexes
DROP INDEX IF EXISTS idx_users_address_id;
DROP INDEX IF EXISTS idx_restaurants_address_id;

-- 4. Drop address_id columns
ALTER TABLE users DROP COLUMN IF EXISTS address_id;
ALTER TABLE restaurants DROP COLUMN IF EXISTS address_id;
