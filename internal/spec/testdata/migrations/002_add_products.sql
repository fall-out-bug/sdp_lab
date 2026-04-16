-- Add products
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    price DECIMAL(10,2) NOT NULL CHECK (price > 0),
    sku VARCHAR(50) UNIQUE,
    description TEXT DEFAULT ''
);

ALTER TABLE orders ADD COLUMN product_id INTEGER REFERENCES products(id);
