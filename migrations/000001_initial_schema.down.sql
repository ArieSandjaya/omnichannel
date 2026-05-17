DROP POLICY IF EXISTS no_delete_inventory ON inventory_log;
DROP POLICY IF EXISTS no_update_inventory ON inventory_log;
DROP POLICY IF EXISTS tenant_isolation_users ON users;
DROP POLICY IF EXISTS tenant_isolation_inventory ON inventory_log;
DROP POLICY IF EXISTS tenant_isolation_orders ON orders;
DROP POLICY IF EXISTS tenant_isolation_products ON products;

DROP FUNCTION IF EXISTS deduct_stock(UUID, UUID, UUID, INTEGER, TEXT);
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS inventory_log CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS product_categories CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "uuid-ossp";
