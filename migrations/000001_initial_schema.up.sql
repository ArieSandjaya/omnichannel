-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- TABLE: tenants
-- ============================================================
CREATE TABLE tenants (
    id                  UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                VARCHAR(255) NOT NULL,
    slug                VARCHAR(100) NOT NULL UNIQUE,
    business_type       VARCHAR(50)  NOT NULL DEFAULT 'retail',
    subscription_plan   VARCHAR(50)  NOT NULL DEFAULT 'starter',
    subscription_status VARCHAR(50)  NOT NULL DEFAULT 'active',
    settings            JSONB        NOT NULL DEFAULT '{}',
    timezone            VARCHAR(50)  NOT NULL DEFAULT 'Asia/Jakarta',
    currency            VARCHAR(10)  NOT NULL DEFAULT 'IDR',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);

-- ============================================================
-- TABLE: users
-- ============================================================
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name     VARCHAR(255),
    role          VARCHAR(50)  NOT NULL DEFAULT 'staff',
    -- owner | manager | cashier | staff | viewer
    phone         VARCHAR(20),
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email  ON users(tenant_id, email);

-- ============================================================
-- TABLE: product_categories
-- ============================================================
CREATE TABLE product_categories (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id  UUID         REFERENCES product_categories(id),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL,
    sort_order INTEGER      NOT NULL DEFAULT 0,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_categories_tenant ON product_categories(tenant_id);

-- ============================================================
-- TABLE: products
-- ============================================================
CREATE TABLE products (
    id             UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category_id    UUID          REFERENCES product_categories(id) ON DELETE SET NULL,
    sku            VARCHAR(100)  NOT NULL,
    barcode        VARCHAR(100),
    name           VARCHAR(255)  NOT NULL,
    description    TEXT,
    price          NUMERIC(15,2) NOT NULL DEFAULT 0,
    cost_price     NUMERIC(15,2),
    stock_quantity INTEGER       NOT NULL DEFAULT 0,
    min_stock_level INTEGER      NOT NULL DEFAULT 0,
    track_inventory BOOLEAN      NOT NULL DEFAULT TRUE,
    unit           VARCHAR(50)   NOT NULL DEFAULT 'pcs',
    weight_grams   INTEGER       NOT NULL DEFAULT 0,
    image_url      TEXT,
    images         JSONB         NOT NULL DEFAULT '[]',
    channels       JSONB         NOT NULL DEFAULT '["pos"]',
    is_active      BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, sku)
);

CREATE INDEX idx_products_tenant    ON products(tenant_id);
CREATE INDEX idx_products_sku       ON products(tenant_id, sku);
CREATE INDEX idx_products_barcode   ON products(tenant_id, barcode) WHERE barcode IS NOT NULL;
CREATE INDEX idx_products_category  ON products(tenant_id, category_id);
CREATE INDEX idx_products_low_stock ON products(tenant_id, stock_quantity)
    WHERE track_inventory = TRUE;
CREATE INDEX idx_products_name      ON products USING gin(name gin_trgm_ops);

-- ============================================================
-- TABLE: orders
-- ============================================================
CREATE TABLE orders (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_number    VARCHAR(100)  NOT NULL,
    channel         VARCHAR(50)   NOT NULL DEFAULT 'pos',
    -- pos | website | shopee | tokopedia | tiktok
    status          VARCHAR(50)   NOT NULL DEFAULT 'pending',
    -- pending | awaiting_payment | paid | processing | shipped | delivered | cancelled
    payment_status  VARCHAR(50)   NOT NULL DEFAULT 'unpaid',
    -- unpaid | pending | paid | expired | failed
    customer_name   VARCHAR(255)  NOT NULL,
    customer_email  VARCHAR(255)  NOT NULL DEFAULT '',
    customer_phone  VARCHAR(20),
    subtotal        NUMERIC(15,2) NOT NULL DEFAULT 0,
    shipping_cost   NUMERIC(15,2) NOT NULL DEFAULT 0,
    total           NUMERIC(15,2) NOT NULL DEFAULT 0,
    notes           TEXT,
    line_items      JSONB         NOT NULL DEFAULT '[]',
    -- [{product_id, sku, name, quantity, price, subtotal}]
    payment_info    JSONB,
    -- {external_id, payment_type, xendit_id, amount, status, qris_string, va_number, bank_code, expires_at, paid_at, tenant_id}
    shipping_info   JSONB,
    -- {biteship_order_id, courier_code, courier_service, tracking_number, waybill_id, status, price}
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, order_number)
);

CREATE INDEX idx_orders_tenant        ON orders(tenant_id);
CREATE INDEX idx_orders_number        ON orders(tenant_id, order_number);
CREATE INDEX idx_orders_status        ON orders(tenant_id, status);
CREATE INDEX idx_orders_payment_status ON orders(tenant_id, payment_status);
CREATE INDEX idx_orders_created_at    ON orders(tenant_id, created_at DESC);

-- Expression indexes for webhook JSONB lookups
CREATE INDEX idx_orders_payment_external_id
    ON orders ((payment_info->>'external_id'))
    WHERE payment_info IS NOT NULL;

CREATE INDEX idx_orders_shipping_waybill
    ON orders ((shipping_info->>'waybill_id'))
    WHERE shipping_info IS NOT NULL;

CREATE INDEX idx_orders_shipping_biteship_id
    ON orders ((shipping_info->>'biteship_order_id'))
    WHERE shipping_info IS NOT NULL;

-- ============================================================
-- TABLE: inventory_log  (append-only, no UPDATE/DELETE)
-- ============================================================
CREATE TABLE inventory_log (
    id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id     UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    order_id       UUID        REFERENCES orders(id) ON DELETE SET NULL,
    change_type    VARCHAR(50) NOT NULL,
    -- deduct | restock | adjustment | initial
    quantity       INTEGER     NOT NULL,
    stock_before   INTEGER     NOT NULL,
    stock_after    INTEGER     NOT NULL,
    reason         TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inv_log_tenant     ON inventory_log(tenant_id);
CREATE INDEX idx_inv_log_product    ON inventory_log(tenant_id, product_id);
CREATE INDEX idx_inv_log_type       ON inventory_log(tenant_id, change_type);
CREATE INDEX idx_inv_log_created_at ON inventory_log(tenant_id, created_at DESC);

-- ============================================================
-- TRIGGER: auto-update updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_products_updated_at
    BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_orders_updated_at
    BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- FUNCTION: deduct_stock  (atomic, uses FOR UPDATE lock)
-- ============================================================
CREATE OR REPLACE FUNCTION deduct_stock(
    p_tenant_id   UUID,
    p_product_id  UUID,
    p_order_id    UUID,
    p_quantity    INTEGER,
    p_reason      TEXT
) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    v_stock_before INTEGER;
    v_stock_after  INTEGER;
BEGIN
    -- Lock baris produk untuk cegah concurrent deduction
    SELECT stock_quantity INTO v_stock_before
    FROM products
    WHERE id = p_product_id AND tenant_id = p_tenant_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Product % not found for tenant %', p_product_id, p_tenant_id;
    END IF;

    IF v_stock_before < p_quantity THEN
        RAISE EXCEPTION 'Insufficient stock for product %. Have %, need %',
            p_product_id, v_stock_before, p_quantity;
    END IF;

    v_stock_after := v_stock_before - p_quantity;

    UPDATE products
    SET stock_quantity = v_stock_after,
        updated_at     = NOW()
    WHERE id = p_product_id AND tenant_id = p_tenant_id;

    INSERT INTO inventory_log
        (tenant_id, product_id, order_id, change_type, quantity, stock_before, stock_after, reason)
    VALUES
        (p_tenant_id, p_product_id, p_order_id, 'deduct', p_quantity,
         v_stock_before, v_stock_after, p_reason);
END;
$$;

-- ============================================================
-- ROW-LEVEL SECURITY
-- ============================================================
ALTER TABLE products      ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders        ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE users         ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_products ON products
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID);

CREATE POLICY tenant_isolation_orders ON orders
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID);

CREATE POLICY tenant_isolation_inventory ON inventory_log
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID);

CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::UUID);

-- inventory_log adalah append-only
CREATE POLICY no_update_inventory ON inventory_log FOR UPDATE USING (FALSE);
CREATE POLICY no_delete_inventory ON inventory_log FOR DELETE USING (FALSE);
