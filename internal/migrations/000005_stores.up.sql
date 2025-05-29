CREATE TABLE stores (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    store_type_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    image VARCHAR(255),
    contact_phone VARCHAR(20) not null,
    contact_email VARCHAR(100),
    address_text TEXT,
    latitude NUMERIC(10, 7),
    longitude NUMERIC(10, 7),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (store_type_id) REFERENCES store_types(id),
    CONSTRAINT address_or_coordinates CHECK (
        (address_text IS NOT NULL AND address_text != '') OR 
        (latitude IS NOT NULL AND longitude IS NOT NULL)
    )
);

CREATE INDEX idx_stores_owner_id ON stores(owner_id);
CREATE INDEX idx_stores_store_type_id ON stores(store_type_id);
