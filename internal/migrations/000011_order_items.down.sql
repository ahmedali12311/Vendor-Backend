CREATE INDEX idx_order_items_order_id ON order_items(order_id);
DROP INDEX IF EXISTS idx_order_items_order_id;

DROP TABLE IF EXISTS order_items CASCADE;
