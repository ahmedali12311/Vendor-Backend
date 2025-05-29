CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (id, name)
VALUES
    (1, 'admin'),
    (2, 'citizen'),
    (3, 'owner'),
    (4, 'delivery')

ON CONFLICT (id) DO NOTHING;