-- 1. Remove the standard index on 'id' (redundant to PRIMARY KEY)
DROP INDEX IF EXISTS idx_users_id;

-- 2. Remove the standard index on 'email' (redundant to UNIQUE CONSTRAINT)
DROP INDEX IF EXISTS idx_users_email;

-- 3. Remove one of the duplicate UNIQUE CONSTRAINTS on 'email'
-- Note: You are dropping the CONSTRAINT, which automatically removes its underlying index.
ALTER TABLE public.users DROP CONSTRAINT IF EXISTS users_email_unique;