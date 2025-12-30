-- Alter restaurants table
-- 1. Add new columns
ALTER TABLE restaurants ADD COLUMN avatar_url VARCHAR(500);
ALTER TABLE restaurants ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE restaurants ADD COLUMN user_id UUID;
ALTER TABLE restaurants ADD COLUMN capacity INTEGER;
ALTER TABLE restaurants ADD COLUMN delivery_available BOOLEAN DEFAULT false;
ALTER TABLE restaurants ADD COLUMN takeaway_available BOOLEAN DEFAULT false;

-- 2. Modify existing columns
ALTER TABLE restaurants ALTER COLUMN id TYPE UUID USING id::uuid;
ALTER TABLE restaurants ALTER COLUMN address TYPE TEXT;

-- 3. Drop cuisine_type column
ALTER TABLE restaurants DROP COLUMN cuisine_type;

-- 4. Add foreign key constraint to user_id
ALTER TABLE restaurants ADD CONSTRAINT fk_restaurant_user 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- 5. Add constraint for status values
ALTER TABLE restaurants ADD CONSTRAINT chk_restaurant_status 
    CHECK (status IN ('active', 'inactive', 'blocked', 'deleted'));
