CREATE TABLE IF NOT EXISTS orders (
    id        SERIAL PRIMARY KEY,
    order_uid  VARCHAR(50) UNIQUE,
    order_data JSONB);
CREATE INDEX IF NOT EXISTS oid ON orders(order_uid);
