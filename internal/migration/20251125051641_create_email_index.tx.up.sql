-- Create an index on the email column for faster lookups during registration
CREATE INDEX IF NOT EXISTS idx_users_email ON public.users (email);
CREATE INDEX IF NOT EXISTS idx_users_id ON public.users (id);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON public.users (created_at);