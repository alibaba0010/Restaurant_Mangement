-- 1. Create the ENUM type first
CREATE TYPE user_status AS ENUM ('active', 'blocked', 'deleted');

-- 2. Alter the table to add all columns at once
ALTER TABLE users 
    ADD COLUMN avatar_url TEXT,
    ADD COLUMN phone_number TEXT,
    ADD COLUMN status user_status DEFAULT 'active' NOT NULL;