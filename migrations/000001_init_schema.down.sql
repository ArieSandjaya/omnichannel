-- Drop in reverse dependency order

DROP FUNCTION IF EXISTS deduct_stock(UUID, UUID, INTEGER, UUID, VARCHAR, UUID);
DROP FUNCTION IF EXISTS current_tenant_id();
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

DROP TABLE IF EXISTS inventory_log      CASCADE;
DROP TABLE IF EXISTS orders             CASCADE;
DROP TABLE IF EXISTS products           CASCADE;
DROP TABLE IF EXISTS product_categories CASCADE;
DROP TABLE IF EXISTS users              CASCADE;
DROP TABLE IF EXISTS tenants            CASCADE;

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "uuid-ossp";
