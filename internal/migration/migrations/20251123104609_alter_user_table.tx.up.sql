ALTER TABLE public.users
ADD address VARCHAR(255),
ADD role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'management', 'admin'));