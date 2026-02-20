CREATE TABLE IF NOT EXISTS menu_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_restaurant FOREIGN KEY (restaurant_id) REFERENCES restaurants(id) ON DELETE CASCADE
);

ALTER TABLE menu ADD COLUMN category_id UUID;
ALTER TABLE menu ADD COLUMN tags JSONB; -- Store tags like ["vegan", "spicy"]
ALTER TABLE menu ADD CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES menu_categories(id) ON DELETE SET NULL;
