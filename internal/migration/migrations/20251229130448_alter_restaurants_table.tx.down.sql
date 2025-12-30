-- Reverse the alterations to restaurants table

-- 1. Remove constraints
ALTER TABLE restaurants DROP CONSTRAINT IF EXISTS chk_restaurant_status;
ALTER TABLE restaurants DROP CONSTRAINT IF EXISTS fk_restaurant_user;

-- 2. Add back cuisine_type column
ALTER TABLE restaurants ADD COLUMN cuisine_type VARCHAR(100);

-- 3. Revert column type changes
ALTER TABLE restaurants ALTER COLUMN address TYPE VARCHAR(200);
ALTER TABLE restaurants ALTER COLUMN id TYPE VARCHAR(36);

-- 4. Drop new columns
ALTER TABLE restaurants DROP COLUMN IF EXISTS takeaway_available;
ALTER TABLE restaurants DROP COLUMN IF EXISTS delivery_available;
ALTER TABLE restaurants DROP COLUMN IF EXISTS capacity;
ALTER TABLE restaurants DROP COLUMN IF EXISTS user_id;
ALTER TABLE restaurants DROP COLUMN IF EXISTS status;
ALTER TABLE restaurants DROP COLUMN IF EXISTS avatar_url;
