DROP INDEX IF EXISTS idx_restaurants_capacity_id;
DROP INDEX IF EXISTS idx_restaurants_rating_id;
DROP INDEX IF EXISTS idx_restaurants_created_at_id;
DROP INDEX IF EXISTS idx_users_email_trgm;
DROP INDEX IF EXISTS idx_users_name_trgm;
DROP INDEX IF EXISTS idx_restaurants_description_trgm;
DROP INDEX IF EXISTS idx_restaurants_name_trgm;
DROP EXTENSION IF EXISTS pg_trgm;
