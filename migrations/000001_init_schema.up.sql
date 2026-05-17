-- ==========================================================
-- EXTENSION
-- ==========================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";


-- ==========================================================
-- TABLE: tenants
-- ==========================================================
CREATE TABLE tenants (
  id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name                VARCHAR(255) NOT NULL,
  slug                VARCHAR(100) NOT NULL UNIQUE,
  business_type       VARCHAR(50)  NOT NULL DEFAULT 'retail',
  subscription_plan   VARCHAR(50)  NOT NULL DEFAULT 'starter',
  subscription_status VARCHAR(50)  NOT NULL DEFAULT 'active',
  trial_ends_at       TIMESTAMPTZ,
  settings            JSONB        NOT NULL DEFAULT '{}',
  logo_url            TEXT,
  timezone            VARCHAR(50)  NOT NULL DEFAULT 'Asia/Jakarta',
  currency            VARCHAR(10)  NOT NULL DEFAULT 'IDR',
  created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug         ON tenants(slug);
CREATE INDEX idx_tenants_subscription ON tenants(subscription_status);


-- ==========================================================
-- TABLE: users
-- ==========================================================
CREATE TABLE users (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  auth_user_id  UUID         UNIQUE,
  email         VARCHAR(255) NOT NULL,
  full_name     VARCHAR(255),
  role          VARCHAR(50)  NOT NULL DEFAULT 'staff',
  avatar_url    TEXT,
  phone         VARCHAR(20),
  is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant    ON users(tenant_id);
CREATE INDEX idx_users_auth_user ON users(auth_user_id);
CREATE INDEX idx_users_email     ON users(tenant_id, email);


-- ==========================================================
-- TABLE: product_categories
-- ==========================================================
CREATE TABLE product_categories (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
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


-- ==========================================================
-- TABLE: products
-- ==========================================================
CREATE TABLE products (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  category_id      UUID          REFERENCES product_categories(id) ON DELETE SET NULL,
  sku              VARCHAR(100)  NOT NULL,
  barcode          VARCHAR(100),
  name             VARCHAR(255)  NOT NULL,
  description      TEXT,
  price            DECIMAL(15,2) NOT NULL DEFAULT 0,
  cost_price       DECIMAL(15,2),
  compare_at_price DECIMAL(15,2),
  stock_quantity   INTEGER       NOT NULL DEFAULT 0,
  min_stock_level  INTEGER       NOT NULL DEFAULT 0,
  track_inventory  BOOLEAN       NOT NULL DEFAULT TRUE,
  unit             VARCHAR(50)   NOT NULL DEFAULT 'pcs',
  weight           DECIMAL(10,3),
  images           JSONB         NOT NULL DEFAULT '[]',
  attributes       JSONB         NOT NULL DEFAULT '{}',
  channels         JSONB         NOT NULL DEFAULT '["pos"]',
  channel_listing  JSONB         NOT NULL DEFAULT '{}',
  is_active        BOOLEAN       NOT NULL DEFAULT TRUE,
  created_by       UUID          REFERENCES users(id),
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, sku)
);

CREATE INDEX idx_products_tenant      ON products(tenant_id);
CREATE INDEX idx_products_sku         ON products(tenant_id, sku);
CREATE INDEX idx_products_barcode     ON products(tenant_id, barcode) WHERE barcode IS NOT NULL;
CREATE INDEX idx_products_category    ON products(tenant_id, category_id);
CREATE INDEX idx_products_low_stock   ON products(tenant_id, stock_quantity)
  WHERE track_inventory = TRUE;
CREATE INDEX idx_products_name_search ON products USING gin(name gin_trgm_ops);


-- ==========================================================
-- TABLE: orders
-- ==========================================================
CREATE TABLE orders (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  order_number     VARCHAR(100)  NOT NULL,
  channel          VARCHAR(50)   NOT NULL,
  channel_order_id VARCHAR(255),
  status           VARCHAR(50)   NOT NULL DEFAULT 'pending',
  customer_id      UUID          REFERENCES users(id),
  customer_info    JSONB         NOT NULL DEFAULT '{}',
  line_items       JSONB         NOT NULL DEFAULT '[]',
  subtotal         DECIMAL(15,2) NOT NULL DEFAULT 0,
  discount_amount  DECIMAL(15,2) NOT NULL DEFAULT 0,
  tax_amount       DECIMAL(15,2) NOT NULL DEFAULT 0,
  shipping_amount  DECIMAL(15,2) NOT NULL DEFAULT 0,
  total_amount     DECIMAL(15,2) NOT NULL DEFAULT 0,
  payment_method   VARCHAR(50),
  payment_status   VARCHAR(50)   NOT NULL DEFAULT 'unpaid',
  payment_info     JSONB         NOT NULL DEFAULT '{}',
  shipping_address JSONB         DEFAULT '{}',
  shipping_info    JSONB         DEFAULT '{}',
  notes            TEXT,
  metadata         JSONB         NOT NULL DEFAULT '{}',
  created_by       UUID          REFERENCES users(id),
  completed_at     TIMESTAMPTZ,
  cancelled_at     TIMESTAMPTZ,
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, order_number)
);

CREATE INDEX idx_orders_tenant     ON orders(tenant_id);
CREATE INDEX idx_orders_number     ON orders(tenant_id, order_number);
CREATE INDEX idx_orders_channel    ON orders(tenant_id, channel);
CREATE INDEX idx_orders_status     ON orders(tenant_id, status);
CREATE INDEX idx_orders_created_at ON orders(tenant_id, created_at DESC);
CREATE INDEX idx_orders_channel_id ON orders(tenant_id, channel, channel_order_id)
  WHERE channel_order_id IS NOT NULL;


-- ==========================================================
-- TABLE: inventory_log
-- ==========================================================
CREATE TABLE inventory_log (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  product_id      UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  type            VARCHAR(50) NOT NULL,
  quantity_change INTEGER     NOT NULL,
  quantity_before INTEGER     NOT NULL,
  quantity_after  INTEGER     NOT NULL,
  reference_type  VARCHAR(50),
  reference_id    UUID,
  channel         VARCHAR(50),
  notes           TEXT,
  created_by      UUID        REFERENCES users(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inv_log_tenant     ON inventory_log(tenant_id);
CREATE INDEX idx_inv_log_product    ON inventory_log(tenant_id, product_id);
CREATE INDEX idx_inv_log_type       ON inventory_log(tenant_id, type);
CREATE INDEX idx_inv_log_created_at ON inventory_log(tenant_id, created_at DESC);
CREATE INDEX idx_inv_log_reference  ON inventory_log(tenant_id, reference_type, reference_id)
  WHERE reference_id IS NOT NULL;


-- ==========================================================
-- TRIGGER: Auto-update updated_at
-- ==========================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenants_updated_at
  BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_products_updated_at
  BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_orders_updated_at
  BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ==========================================================
-- ROW-LEVEL SECURITY
-- ==========================================================
ALTER TABLE tenants            ENABLE ROW LEVEL SECURITY;
ALTER TABLE users              ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE products           ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders             ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_log      ENABLE ROW LEVEL SECURITY;

-- Extract tenant_id from JWT claims (used by Supabase PostgREST)
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
  SELECT COALESCE(
    (current_setting('request.jwt.claims', true)::jsonb ->> 'tenant_id')::UUID,
    NULL
  );
$$ LANGUAGE SQL STABLE;

CREATE POLICY users_tenant_isolation ON users
  FOR ALL USING (tenant_id = current_tenant_id());

CREATE POLICY categories_tenant_isolation ON product_categories
  FOR ALL USING (tenant_id = current_tenant_id());

CREATE POLICY products_tenant_isolation ON products
  FOR ALL USING (tenant_id = current_tenant_id());

CREATE POLICY products_public_read ON products
  FOR SELECT USING (
    is_active = TRUE
    AND 'website' = ANY(SELECT jsonb_array_elements_text(channels))
  );

CREATE POLICY orders_tenant_isolation ON orders
  FOR ALL USING (tenant_id = current_tenant_id());

CREATE POLICY inventory_log_tenant_isolation ON inventory_log
  FOR ALL USING (tenant_id = current_tenant_id());

-- Immutable audit trail: no UPDATE or DELETE on inventory_log
CREATE POLICY inventory_log_no_update ON inventory_log FOR UPDATE USING (FALSE);
CREATE POLICY inventory_log_no_delete ON inventory_log FOR DELETE USING (FALSE);


-- ==========================================================
-- FUNCTION: Atomic stock deduction with audit log
-- Used as alternative to application-level SELECT FOR UPDATE
-- ==========================================================
CREATE OR REPLACE FUNCTION deduct_stock(
  p_tenant_id    UUID,
  p_product_id   UUID,
  p_quantity     INTEGER,
  p_reference_id UUID,
  p_channel      VARCHAR(50),
  p_user_id      UUID
)
RETURNS TABLE(success BOOLEAN, message TEXT, new_quantity INTEGER)
LANGUAGE plpgsql AS $$
DECLARE
  v_before INTEGER;
  v_after  INTEGER;
BEGIN
  SELECT stock_quantity INTO v_before
  FROM products
  WHERE id = p_product_id AND tenant_id = p_tenant_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RETURN QUERY SELECT FALSE, 'Produk tidak ditemukan'::TEXT, 0;
    RETURN;
  END IF;

  IF v_before < p_quantity THEN
    RETURN QUERY SELECT FALSE, 'Stok tidak mencukupi'::TEXT, v_before;
    RETURN;
  END IF;

  v_after := v_before - p_quantity;

  UPDATE products
  SET stock_quantity = v_after, updated_at = NOW()
  WHERE id = p_product_id AND tenant_id = p_tenant_id;

  INSERT INTO inventory_log (
    tenant_id, product_id, type, quantity_change,
    quantity_before, quantity_after, reference_type,
    reference_id, channel, created_by
  ) VALUES (
    p_tenant_id, p_product_id, 'sale', -p_quantity,
    v_before, v_after, 'order',
    p_reference_id, p_channel, p_user_id
  );

  RETURN QUERY SELECT TRUE, 'OK'::TEXT, v_after;
END;
$$;
