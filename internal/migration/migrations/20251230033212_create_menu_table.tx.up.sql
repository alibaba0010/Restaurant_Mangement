CREATE TABLE IF NOT EXISTS menu (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    image_urls JSONB,
    video_url VARCHAR(500),
    restaurant_id UUID NOT NULL,
    is_available BOOLEAN DEFAULT TRUE,
    prep_time_minutes INT,
    calories INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_restaurant
        FOREIGN KEY (restaurant_id)
        REFERENCES restaurants(id)
        ON DELETE CASCADE
);
