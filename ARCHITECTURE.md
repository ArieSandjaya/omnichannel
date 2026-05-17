# Arsitektur SaaS Omnichannel POS + E-commerce

> **Versi:** 1.0.0  
> **Tanggal:** 2026-05-17  
> **Status:** Draft — System Design

---

## Daftar Isi

1. [Ringkasan Arsitektur](#1-ringkasan-arsitektur)
2. [Strategi Multi-Tenancy](#2-strategi-multi-tenancy)
3. [Stack Teknologi](#3-stack-teknologi)
4. [Struktur Folder — Backend](#4-struktur-folder--backend)
5. [Struktur Folder — Frontend](#5-struktur-folder--frontend)
6. [Skema Database](#6-skema-database)
7. [Row-Level Security (RLS)](#7-row-level-security-rls)
8. [Sinkronisasi Stok Real-time](#8-sinkronisasi-stok-real-time)
9. [Alur Autentikasi & Otorisasi](#9-alur-autentikasi--otorisasi)
10. [Deployment & Infrastruktur](#10-deployment--infrastruktur)

---

## 1. Ringkasan Arsitektur

Platform ini adalah **SaaS multi-tenant** yang memungkinkan banyak toko (tenant) beroperasi dalam satu database PostgreSQL secara aman dan terisolasi. Setiap tenant memiliki:

- Manajemen produk & stok terpusat
- Titik penjualan (POS) di toko fisik
- Storefront e-commerce online
- Integrasi marketplace (Shopee, Tokopedia, TikTok Shop)
- Sinkronisasi stok real-time lintas channel

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                                  │
│   Next.js Dashboard  │  POS Terminal  │  Mobile App  │  Storefront  │
└────────────────┬────────────────────────────────┬────────────────────┘
                 │ HTTPS / WebSocket               │ HTTPS
┌────────────────▼─────────────────────────────────▼────────────────────┐
│                         API GATEWAY (NestJS)                           │
│   Auth Middleware  │  Tenant Resolver  │  Rate Limiter  │  Logger      │
└─────────┬───────────────────┬──────────────────┬─────────────────────┘
          │                   │                  │
┌─────────▼──────┐  ┌─────────▼──────┐  ┌───────▼──────────┐
│  Core Modules  │  │  Channel Sync  │  │  Realtime Engine  │
│  - Auth        │  │  - Shopee      │  │  - Supabase RT    │
│  - Products    │  │  - Tokopedia   │  │  - Redis PubSub   │
│  - Orders      │  │  - TikTok Shop │  │  - WebSocket      │
│  - Inventory   │  │  - Website     │  └──────────────────┘
└─────────┬──────┘  └─────────┬──────┘
          │                   │
┌─────────▼───────────────────▼─────────┐
│          PostgreSQL (Supabase)          │
│   Multi-tenant dengan Row-Level        │
│   Security (RLS) per tenant_id         │
└───────────────────────────────────────┘
```

---

## 2. Strategi Multi-Tenancy

### Pendekatan: Shared Database + Shared Schema + Row-Level Security

Dari tiga pendekatan multi-tenancy yang umum, dipilih **shared schema dengan RLS** karena:

| Aspek | Separate DB | Separate Schema | Shared Schema + RLS |
|-------|-------------|-----------------|---------------------|
| Isolasi data | Sangat tinggi | Tinggi | Tinggi (via RLS) |
| Biaya infrastruktur | Tinggi | Sedang | Rendah |
| Kompleksitas operasional | Tinggi | Sedang | Rendah |
| Skalabilitas tenant baru | Lambat | Sedang | Instan |
| Cocok untuk | Enterprise | Mid-market | SMB / Growth SaaS |

### Mekanisme Isolasi

1. Setiap tabel memiliki kolom `tenant_id UUID NOT NULL`
2. RLS Policy memfilter semua query otomatis berdasarkan `tenant_id` dari JWT claim
3. Application layer selalu menyertakan `tenant_id` di setiap query
4. Indeks komposit `(tenant_id, ...)` memastikan performa optimal

```
JWT Token
  └── claims.tenant_id  ──►  RLS Policy  ──►  Data tenant A saja
                                               (tenant B tidak terlihat)
```

---

## 3. Stack Teknologi

### Backend
| Komponen | Teknologi | Alasan |
|----------|-----------|--------|
| Framework | NestJS (Node.js + TypeScript) | Modular, DI, scalable |
| Database | PostgreSQL via Supabase | RLS native, Realtime built-in |
| ORM | Prisma | Type-safe, migration tooling |
| Auth | Supabase Auth + JWT | Integrasi RLS seamless |
| Cache | Redis (Upstash) | Session, rate limit, pub/sub |
| Queue | BullMQ | Async job processing |
| Realtime | Supabase Realtime + Socket.io | Stock sync, POS updates |
| File Storage | Supabase Storage | Gambar produk |
| API Docs | Swagger / OpenAPI 3.0 | Developer experience |

### Frontend
| Komponen | Teknologi | Alasan |
|----------|-----------|--------|
| Framework | Next.js 14 (App Router) | SSR, SSG, API routes |
| Language | TypeScript | Type safety |
| Styling | Tailwind CSS + shadcn/ui | Konsisten, cepat |
| State | Zustand | Lightweight, simple |
| Data fetching | TanStack Query | Caching, sync server state |
| Realtime | Supabase JS client | Auto-reconnect, multiplexed |
| Forms | React Hook Form + Zod | Validasi client+server |
| POS UI | Custom (touch-optimized) | Kebutuhan kasir spesifik |

---

## 4. Struktur Folder — Backend

```
backend/
├── src/
│   ├── app.module.ts
│   ├── main.ts
│   │
│   ├── config/                        # Konfigurasi aplikasi
│   │   ├── app.config.ts
│   │   ├── database.config.ts
│   │   ├── redis.config.ts
│   │   └── supabase.config.ts
│   │
│   ├── common/                        # Shared utilities
│   │   ├── decorators/
│   │   │   ├── current-tenant.decorator.ts
│   │   │   └── current-user.decorator.ts
│   │   ├── filters/
│   │   │   └── http-exception.filter.ts
│   │   ├── guards/
│   │   │   ├── jwt-auth.guard.ts
│   │   │   └── roles.guard.ts
│   │   ├── interceptors/
│   │   │   ├── tenant.interceptor.ts    # Inject tenant_id ke setiap request
│   │   │   └── transform.interceptor.ts
│   │   ├── pipes/
│   │   │   └── validation.pipe.ts
│   │   └── types/
│   │       ├── tenant-context.type.ts
│   │       └── paginated-result.type.ts
│   │
│   ├── database/
│   │   ├── prisma.service.ts           # Prisma client singleton
│   │   ├── migrations/                 # SQL migration files
│   │   └── seeds/                      # Seed data untuk development
│   │
│   ├── modules/
│   │   ├── auth/                       # Autentikasi & otorisasi
│   │   │   ├── auth.module.ts
│   │   │   ├── auth.controller.ts
│   │   │   ├── auth.service.ts
│   │   │   ├── strategies/
│   │   │   │   └── jwt.strategy.ts
│   │   │   └── dto/
│   │   │       ├── login.dto.ts
│   │   │       └── register.dto.ts
│   │   │
│   │   ├── tenants/                    # Manajemen tenant/toko
│   │   │   ├── tenants.module.ts
│   │   │   ├── tenants.controller.ts
│   │   │   ├── tenants.service.ts
│   │   │   └── dto/
│   │   │       ├── create-tenant.dto.ts
│   │   │       └── update-tenant.dto.ts
│   │   │
│   │   ├── users/                      # Manajemen pengguna per tenant
│   │   │   ├── users.module.ts
│   │   │   ├── users.controller.ts
│   │   │   ├── users.service.ts
│   │   │   └── dto/
│   │   │
│   │   ├── products/                   # Katalog produk
│   │   │   ├── products.module.ts
│   │   │   ├── products.controller.ts
│   │   │   ├── products.service.ts
│   │   │   ├── categories/
│   │   │   └── dto/
│   │   │
│   │   ├── orders/                     # Pemrosesan pesanan
│   │   │   ├── orders.module.ts
│   │   │   ├── orders.controller.ts
│   │   │   ├── orders.service.ts
│   │   │   ├── pos-orders/             # Alur khusus POS
│   │   │   └── dto/
│   │   │
│   │   ├── inventory/                  # Manajemen stok
│   │   │   ├── inventory.module.ts
│   │   │   ├── inventory.controller.ts
│   │   │   ├── inventory.service.ts
│   │   │   ├── inventory-log.service.ts
│   │   │   └── dto/
│   │   │
│   │   ├── channels/                   # Integrasi sales channel
│   │   │   ├── channels.module.ts
│   │   │   ├── channels.service.ts     # Channel abstraction
│   │   │   ├── pos/
│   │   │   │   └── pos.service.ts
│   │   │   ├── shopee/
│   │   │   │   ├── shopee.service.ts
│   │   │   │   └── shopee-webhook.controller.ts
│   │   │   ├── tokopedia/
│   │   │   │   ├── tokopedia.service.ts
│   │   │   │   └── tokopedia-webhook.controller.ts
│   │   │   └── website/
│   │   │       └── website.service.ts
│   │   │
│   │   ├── sync/                       # Orkestrasi sinkronisasi
│   │   │   ├── sync.module.ts
│   │   │   ├── sync.service.ts         # Koordinasi stock sync lintas channel
│   │   │   ├── sync.processor.ts       # BullMQ job processor
│   │   │   └── sync.gateway.ts         # WebSocket gateway
│   │   │
│   │   └── webhooks/                   # Inbound webhook handling
│   │       ├── webhooks.module.ts
│   │       └── webhooks.controller.ts
│   │
│   └── jobs/                           # Background jobs
│       ├── stock-sync.job.ts
│       ├── order-fulfillment.job.ts
│       └── report-generation.job.ts
│
├── prisma/
│   ├── schema.prisma
│   └── migrations/
│
├── test/
│   ├── unit/
│   └── e2e/
│
├── .env.example
├── nest-cli.json
├── package.json
└── tsconfig.json
```

---

## 5. Struktur Folder — Frontend

```
frontend/
├── app/                               # Next.js App Router
│   ├── layout.tsx                     # Root layout
│   ├── (auth)/                        # Route group: tanpa sidebar
│   │   ├── layout.tsx
│   │   ├── login/
│   │   │   └── page.tsx
│   │   ├── register/
│   │   │   └── page.tsx
│   │   └── forgot-password/
│   │       └── page.tsx
│   │
│   └── (dashboard)/                   # Route group: dengan sidebar
│       ├── layout.tsx                 # Dashboard shell + sidebar
│       ├── page.tsx                   # Overview / home
│       │
│       ├── pos/                       # Kasir / Point of Sale
│       │   ├── page.tsx               # POS terminal utama
│       │   ├── sessions/              # Sesi kasir (buka/tutup)
│       │   └── receipts/
│       │
│       ├── products/                  # Manajemen produk
│       │   ├── page.tsx               # Daftar produk
│       │   ├── new/
│       │   │   └── page.tsx
│       │   ├── [id]/
│       │   │   ├── page.tsx
│       │   │   └── edit/
│       │   └── categories/
│       │
│       ├── orders/                    # Daftar & detail pesanan
│       │   ├── page.tsx
│       │   └── [id]/
│       │       └── page.tsx
│       │
│       ├── inventory/                 # Stok & log perubahan
│       │   ├── page.tsx
│       │   ├── adjustments/           # Penyesuaian stok manual
│       │   └── log/
│       │
│       ├── channels/                  # Integrasi marketplace
│       │   ├── page.tsx               # Status semua channel
│       │   ├── shopee/
│       │   ├── tokopedia/
│       │   └── website/
│       │
│       ├── reports/                   # Laporan penjualan
│       │   ├── page.tsx
│       │   ├── sales/
│       │   └── inventory/
│       │
│       └── settings/                  # Pengaturan toko
│           ├── page.tsx
│           ├── profile/
│           ├── team/
│           └── billing/
│
├── components/
│   ├── ui/                            # shadcn/ui base components
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   ├── table.tsx
│   │   └── ...                        # (generated by shadcn)
│   │
│   ├── pos/                           # Komponen khusus POS
│   │   ├── ProductGrid.tsx            # Grid produk untuk kasir
│   │   ├── CartPanel.tsx              # Panel keranjang belanja
│   │   ├── PaymentModal.tsx           # Modal proses pembayaran
│   │   ├── ReceiptPrint.tsx           # Struk cetak
│   │   └── StockBadge.tsx             # Badge stok real-time
│   │
│   ├── products/
│   │   ├── ProductForm.tsx
│   │   ├── ProductTable.tsx
│   │   └── StockAlert.tsx             # Alert stok menipis
│   │
│   ├── inventory/
│   │   ├── InventoryLogTable.tsx
│   │   └── AdjustmentForm.tsx
│   │
│   ├── layout/
│   │   ├── Sidebar.tsx
│   │   ├── Header.tsx
│   │   └── MobileNav.tsx
│   │
│   └── shared/
│       ├── DataTable.tsx              # Reusable table dengan pagination
│       ├── ConfirmDialog.tsx
│       ├── LoadingSpinner.tsx
│       └── EmptyState.tsx
│
├── hooks/
│   ├── useRealtimeStock.ts            # Subscribe stok real-time via Supabase
│   ├── useTenant.ts                   # Akses tenant context
│   ├── usePermissions.ts              # Role-based UI guard
│   └── useOfflineCart.ts             # Cart persistence (POS offline mode)
│
├── lib/
│   ├── supabase/
│   │   ├── client.ts                  # Supabase browser client
│   │   ├── server.ts                  # Supabase server client (SSR)
│   │   └── middleware.ts              # Auth middleware
│   ├── api/
│   │   ├── products.ts                # API calls untuk produk
│   │   ├── orders.ts
│   │   └── inventory.ts
│   └── utils/
│       ├── currency.ts                # Format Rupiah
│       ├── date.ts
│       └── validators.ts
│
├── stores/                            # Zustand global state
│   ├── cart.store.ts                  # State keranjang POS
│   ├── tenant.store.ts                # Data tenant aktif
│   └── ui.store.ts                    # UI state (sidebar, modal)
│
├── types/
│   ├── database.ts                    # Auto-generated dari Supabase
│   ├── api.ts
│   └── pos.ts
│
├── middleware.ts                      # Next.js middleware (auth check)
├── next.config.ts
├── tailwind.config.ts
└── package.json
```

---

## 6. Skema Database

### Diagram Relasi

```
tenants (1) ──────────────── (N) users
   │                                │
   │                                │ created_by
   ├──── (N) products               │
   │           │                    ▼
   │           │          orders (N) ──── (1) tenants
   │           │              │
   │           └──── (N) inventory_log ◄──── order/adjustment
   │
   └──── (N) product_categories
```

### DDL Lengkap

```sql
-- ==========================================================
-- EXTENSION
-- ==========================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- Untuk full-text search produk


-- ==========================================================
-- TABLE: tenants
-- Satu baris = satu toko/klien SaaS
-- ==========================================================
CREATE TABLE tenants (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name               VARCHAR(255)  NOT NULL,
  slug               VARCHAR(100)  NOT NULL UNIQUE,          -- URL identifier: toko-abc
  business_type      VARCHAR(50)   NOT NULL DEFAULT 'retail', -- retail, restaurant, service
  subscription_plan  VARCHAR(50)   NOT NULL DEFAULT 'starter', -- starter, growth, enterprise
  subscription_status VARCHAR(50)  NOT NULL DEFAULT 'active', -- active, suspended, cancelled
  trial_ends_at      TIMESTAMPTZ,
  settings           JSONB         NOT NULL DEFAULT '{}',    -- konfigurasi per-toko
  logo_url           TEXT,
  timezone           VARCHAR(50)   NOT NULL DEFAULT 'Asia/Jakarta',
  currency           VARCHAR(10)   NOT NULL DEFAULT 'IDR',
  created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_subscription ON tenants(subscription_status);

-- Contoh settings JSON:
-- {
--   "tax_rate": 11,
--   "low_stock_threshold": 5,
--   "receipt_footer": "Terima kasih!",
--   "channels_enabled": ["pos", "website", "shopee"]
-- }


-- ==========================================================
-- TABLE: users
-- Pengguna yang terikat ke satu tenant
-- ==========================================================
CREATE TABLE users (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  auth_user_id UUID         UNIQUE,                          -- FK ke Supabase auth.users
  email        VARCHAR(255) NOT NULL,
  full_name    VARCHAR(255),
  role         VARCHAR(50)  NOT NULL DEFAULT 'staff',
    -- ENUM: super_admin | owner | manager | cashier | staff | viewer
  avatar_url   TEXT,
  phone        VARCHAR(20),
  is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant    ON users(tenant_id);
CREATE INDEX idx_users_auth_user ON users(auth_user_id);
CREATE INDEX idx_users_email     ON users(tenant_id, email);


-- ==========================================================
-- TABLE: product_categories
-- Kategori produk per tenant
-- ==========================================================
CREATE TABLE product_categories (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  parent_id   UUID         REFERENCES product_categories(id),
  name        VARCHAR(255) NOT NULL,
  slug        VARCHAR(255) NOT NULL,
  sort_order  INTEGER      NOT NULL DEFAULT 0,
  is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_categories_tenant ON product_categories(tenant_id);


-- ==========================================================
-- TABLE: products
-- Katalog produk per tenant
-- ==========================================================
CREATE TABLE products (
  id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  category_id       UUID          REFERENCES product_categories(id) ON DELETE SET NULL,
  sku               VARCHAR(100)  NOT NULL,
  barcode           VARCHAR(100),
  name              VARCHAR(255)  NOT NULL,
  description       TEXT,
  price             DECIMAL(15,2) NOT NULL DEFAULT 0,
  cost_price        DECIMAL(15,2),
  compare_at_price  DECIMAL(15,2),                           -- Harga coret
  stock_quantity    INTEGER       NOT NULL DEFAULT 0,
  min_stock_level   INTEGER       NOT NULL DEFAULT 0,        -- Threshold alert
  track_inventory   BOOLEAN       NOT NULL DEFAULT TRUE,
  unit              VARCHAR(50)   NOT NULL DEFAULT 'pcs',
  weight            DECIMAL(10,3),                          -- Untuk ongkir (kg)
  images            JSONB         NOT NULL DEFAULT '[]',    -- [{url, alt, is_primary}]
  attributes        JSONB         NOT NULL DEFAULT '{}',    -- {color, size, ...}
  channels          JSONB         NOT NULL DEFAULT '["pos"]',
    -- Sales channels: pos, website, shopee, tokopedia, tiktok_shop
  channel_listing   JSONB         NOT NULL DEFAULT '{}',
    -- Mapping ke ID produk di masing-masing marketplace
    -- {"shopee": "123456", "tokopedia": "789"}
  is_active         BOOLEAN       NOT NULL DEFAULT TRUE,
  created_by        UUID          REFERENCES users(id),
  created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
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
-- Transaksi penjualan dari semua channel
-- ==========================================================
CREATE TABLE orders (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  order_number     VARCHAR(100)  NOT NULL,
  channel          VARCHAR(50)   NOT NULL,
    -- ENUM: pos | website | shopee | tokopedia | tiktok_shop | manual
  channel_order_id VARCHAR(255),                             -- ID pesanan di channel asal
  status           VARCHAR(50)   NOT NULL DEFAULT 'pending',
    -- pending | confirmed | processing | shipped | completed | cancelled | refunded
  customer_id      UUID          REFERENCES users(id),
  customer_info    JSONB         NOT NULL DEFAULT '{}',
    -- {name, email, phone, address} — snapshot saat order
  line_items       JSONB         NOT NULL DEFAULT '[]',
    -- [{
    --   product_id, sku, name, quantity,
    --   unit_price, discount, subtotal,
    --   snapshot: {full product data}
    -- }]
  subtotal         DECIMAL(15,2) NOT NULL DEFAULT 0,
  discount_amount  DECIMAL(15,2) NOT NULL DEFAULT 0,
  tax_amount       DECIMAL(15,2) NOT NULL DEFAULT 0,
  shipping_amount  DECIMAL(15,2) NOT NULL DEFAULT 0,
  total_amount     DECIMAL(15,2) NOT NULL DEFAULT 0,
  payment_method   VARCHAR(50),
    -- cash | card | qris | transfer | cod | online
  payment_status   VARCHAR(50)   NOT NULL DEFAULT 'unpaid',
    -- unpaid | paid | partial | refunded
  payment_info     JSONB         NOT NULL DEFAULT '{}',      -- detail transaksi payment
  shipping_address JSONB         DEFAULT '{}',
  shipping_info    JSONB         DEFAULT '{}',               -- kurir, resi, dll
  notes            TEXT,
  metadata         JSONB         NOT NULL DEFAULT '{}',      -- data tambahan per-channel
  created_by       UUID          REFERENCES users(id),
  completed_at     TIMESTAMPTZ,
  cancelled_at     TIMESTAMPTZ,
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, order_number)
);

CREATE INDEX idx_orders_tenant      ON orders(tenant_id);
CREATE INDEX idx_orders_number      ON orders(tenant_id, order_number);
CREATE INDEX idx_orders_channel     ON orders(tenant_id, channel);
CREATE INDEX idx_orders_status      ON orders(tenant_id, status);
CREATE INDEX idx_orders_created_at  ON orders(tenant_id, created_at DESC);
CREATE INDEX idx_orders_channel_id  ON orders(tenant_id, channel, channel_order_id)
  WHERE channel_order_id IS NOT NULL;


-- ==========================================================
-- TABLE: inventory_log
-- Audit trail setiap perubahan stok
-- ==========================================================
CREATE TABLE inventory_log (
  id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  product_id       UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  type             VARCHAR(50) NOT NULL,
    -- sale | purchase | adjustment | transfer_in | transfer_out | return | correction
  quantity_change  INTEGER     NOT NULL,                     -- Negatif = stok keluar
  quantity_before  INTEGER     NOT NULL,
  quantity_after   INTEGER     NOT NULL,
  reference_type   VARCHAR(50),                             -- order | purchase_order | manual
  reference_id     UUID,                                    -- FK ke tabel asal
  channel          VARCHAR(50),                             -- Channel yang memicu perubahan
  notes            TEXT,
  created_by       UUID        REFERENCES users(id),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
  -- Tidak ada updated_at karena log immutable
);

CREATE INDEX idx_inv_log_tenant      ON inventory_log(tenant_id);
CREATE INDEX idx_inv_log_product     ON inventory_log(tenant_id, product_id);
CREATE INDEX idx_inv_log_type        ON inventory_log(tenant_id, type);
CREATE INDEX idx_inv_log_created_at  ON inventory_log(tenant_id, created_at DESC);
CREATE INDEX idx_inv_log_reference   ON inventory_log(tenant_id, reference_type, reference_id)
  WHERE reference_id IS NOT NULL;


-- ==========================================================
-- TRIGGER: Auto-update updated_at
-- ==========================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenants_updated_at
  BEFORE UPDATE ON tenants
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_products_updated_at
  BEFORE UPDATE ON products
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_orders_updated_at
  BEFORE UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## 7. Row-Level Security (RLS)

RLS memastikan bahwa **query apapun** otomatis difilter hanya untuk `tenant_id` yang sesuai dengan JWT token pengguna yang sedang login.

```sql
-- ==========================================================
-- Aktifkan RLS pada semua tabel
-- ==========================================================
ALTER TABLE tenants         ENABLE ROW LEVEL SECURITY;
ALTER TABLE users           ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE products        ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders          ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_log   ENABLE ROW LEVEL SECURITY;


-- ==========================================================
-- Helper function: ekstrak tenant_id dari JWT
-- ==========================================================
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
  SELECT COALESCE(
    (current_setting('request.jwt.claims', true)::jsonb ->> 'tenant_id')::UUID,
    NULL
  );
$$ LANGUAGE SQL STABLE;


-- ==========================================================
-- RLS Policies: users
-- ==========================================================
CREATE POLICY users_tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_tenant_id());


-- ==========================================================
-- RLS Policies: products
-- ==========================================================
CREATE POLICY products_tenant_isolation ON products
  FOR ALL
  USING (tenant_id = current_tenant_id());

-- Storefront publik: produk aktif bisa dibaca tanpa auth
CREATE POLICY products_public_read ON products
  FOR SELECT
  USING (
    is_active = TRUE
    AND 'website' = ANY(SELECT jsonb_array_elements_text(channels))
  );


-- ==========================================================
-- RLS Policies: orders
-- ==========================================================
CREATE POLICY orders_tenant_isolation ON orders
  FOR ALL
  USING (tenant_id = current_tenant_id());

-- Staff hanya bisa lihat order yang mereka buat
CREATE POLICY orders_staff_own ON orders
  FOR SELECT
  USING (
    tenant_id = current_tenant_id()
    AND (
      -- Manager ke atas bisa lihat semua
      (SELECT role FROM users WHERE auth_user_id = auth.uid()) IN ('owner', 'manager')
      -- Kasir hanya bisa lihat ordernya sendiri
      OR created_by = (SELECT id FROM users WHERE auth_user_id = auth.uid())
    )
  );


-- ==========================================================
-- RLS Policies: inventory_log
-- ==========================================================
CREATE POLICY inventory_log_tenant_isolation ON inventory_log
  FOR ALL
  USING (tenant_id = current_tenant_id());

-- Log immutable: tidak ada UPDATE/DELETE
CREATE POLICY inventory_log_no_update ON inventory_log
  FOR UPDATE USING (FALSE);

CREATE POLICY inventory_log_no_delete ON inventory_log
  FOR DELETE USING (FALSE);
```

---

## 8. Sinkronisasi Stok Real-time

### Arsitektur Sync

Masalah utama: ketika ada penjualan di Shopee, stok di POS dan Tokopedia harus ikut terupdate dalam hitungan detik.

```
[Shopee Webhook]
       │ POST /webhooks/shopee
       ▼
[Webhook Controller]
       │ Validasi signature + parse payload
       ▼
[Sync Service]
       │ 1. Simpan order ke database
       │ 2. DEDUCT stock (atomic update)
       │ 3. Tulis inventory_log
       ▼
[PostgreSQL Trigger / Supabase Realtime]
       │ products.stock_quantity berubah
       ▼
[Supabase Realtime Channel]
       │ Broadcast ke semua subscriber tenant tersebut
       ├──► POS Terminal (Supabase JS client)
       └──► Dashboard Web (Supabase JS client)

[BullMQ Job Queue]
       │ Push job: sync-to-channels
       ▼
[Sync Processor]
       ├──► Update listing stok di Shopee API
       ├──► Update listing stok di Tokopedia API
       └──► Update listing stok di TikTok Shop API
```

### Atomic Stock Deduction

Menggunakan PostgreSQL transaction + `FOR UPDATE` untuk mencegah race condition ketika dua channel memesan produk yang sama secara bersamaan:

```sql
-- Prosedur deduct stok yang aman dari race condition
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
  v_current_stock INTEGER;
  v_new_stock     INTEGER;
BEGIN
  -- Lock baris produk agar tidak ada concurrent deduction
  SELECT stock_quantity INTO v_current_stock
  FROM products
  WHERE id = p_product_id AND tenant_id = p_tenant_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RETURN QUERY SELECT FALSE, 'Produk tidak ditemukan'::TEXT, 0;
    RETURN;
  END IF;

  IF v_current_stock < p_quantity THEN
    RETURN QUERY SELECT FALSE, 'Stok tidak mencukupi'::TEXT, v_current_stock;
    RETURN;
  END IF;

  v_new_stock := v_current_stock - p_quantity;

  -- Update stok
  UPDATE products
  SET stock_quantity = v_new_stock, updated_at = NOW()
  WHERE id = p_product_id AND tenant_id = p_tenant_id;

  -- Catat log
  INSERT INTO inventory_log (
    tenant_id, product_id, type,
    quantity_change, quantity_before, quantity_after,
    reference_type, reference_id, channel, created_by
  ) VALUES (
    p_tenant_id, p_product_id, 'sale',
    -p_quantity, v_current_stock, v_new_stock,
    'order', p_reference_id, p_channel, p_user_id
  );

  RETURN QUERY SELECT TRUE, 'Stok berhasil dikurangi'::TEXT, v_new_stock;
END;
$$;
```

### Supabase Realtime — Subscribe di Frontend

```typescript
// hooks/useRealtimeStock.ts
import { useEffect } from 'react'
import { supabase } from '@/lib/supabase/client'
import { useCartStore } from '@/stores/cart.store'

export function useRealtimeStock(tenantId: string) {
  const updateProductStock = useCartStore(s => s.updateProductStock)

  useEffect(() => {
    const channel = supabase
      .channel(`stock-updates-${tenantId}`)
      .on(
        'postgres_changes',
        {
          event: 'UPDATE',
          schema: 'public',
          table: 'products',
          filter: `tenant_id=eq.${tenantId}`,
        },
        (payload) => {
          updateProductStock(
            payload.new.id,
            payload.new.stock_quantity
          )
        }
      )
      .subscribe()

    return () => { supabase.removeChannel(channel) }
  }, [tenantId])
}
```

### BullMQ — Push Sync ke Marketplace

```typescript
// modules/sync/sync.processor.ts
@Processor('stock-sync')
export class SyncProcessor {
  @Process('broadcast-stock')
  async broadcastStock(job: Job<StockSyncPayload>) {
    const { tenantId, productId, newQuantity, channels } = job.data

    const results = await Promise.allSettled(
      channels.map(channel => {
        switch (channel) {
          case 'shopee':    return this.shopee.updateStock(tenantId, productId, newQuantity)
          case 'tokopedia': return this.tokopedia.updateStock(tenantId, productId, newQuantity)
          case 'tiktok':   return this.tiktok.updateStock(tenantId, productId, newQuantity)
        }
      })
    )

    // Log channel yang gagal untuk retry
    results.forEach((result, i) => {
      if (result.status === 'rejected') {
        this.logger.error(`Sync gagal ke ${channels[i]}: ${result.reason}`)
      }
    })
  }
}
```

---

## 9. Alur Autentikasi & Otorisasi

```
1. User login  →  Supabase Auth (email/password / OAuth)
2. Supabase mengembalikan JWT yang sudah di-embed tenant_id
3. Setiap request API menyertakan JWT di header Authorization
4. NestJS JwtStrategy memverifikasi token + ekstrak tenant_id
5. TenantInterceptor meng-inject tenant_id ke request context
6. Service layer menyertakan tenant_id di setiap query
7. PostgreSQL RLS memvalidasi tenant_id sekali lagi sebagai lapisan keamanan ke-2
```

### Role Hierarchy

```
super_admin   → Akses semua tenant (internal platform admin)
    └── owner         → Akses penuh ke satu tenant
          └── manager       → Akses operasional (tidak bisa billing)
                └── cashier       → Hanya akses POS
                      └── staff         → Akses terbatas
                            └── viewer        → Read-only
```

---

## 10. Deployment & Infrastruktur

```
┌──────────────────────────────────────────────────────────┐
│                    Production Stack                        │
├─────────────────┬────────────────────┬────────────────────┤
│   Frontend      │    Backend         │    Data Layer      │
│   Vercel        │    Railway / Fly.io│    Supabase (PaaS) │
│   (Next.js)     │    (NestJS)        │    PostgreSQL +    │
│                 │                    │    Realtime +      │
│                 │    Redis (Upstash) │    Auth + Storage  │
└─────────────────┴────────────────────┴────────────────────┘
```

### Environment Variables (Backend)

```env
# Supabase
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...   # Hanya di backend, bypass RLS jika perlu

# Database
DATABASE_URL=postgresql://...

# Redis
REDIS_URL=rediss://...

# Marketplace APIs
SHOPEE_PARTNER_ID=...
SHOPEE_PARTNER_KEY=...
TOKOPEDIA_CLIENT_ID=...
TOKOPEDIA_CLIENT_SECRET=...

# JWT
JWT_SECRET=...
JWT_EXPIRES_IN=7d
```

### Skalabilitas

| Fase | Tenant | Strategi |
|------|--------|----------|
| MVP (0–100) | Shared infra | 1 Supabase project, 1 Railway instance |
| Growth (100–1000) | Shared + read replica | Connection pooling (PgBouncer), Redis cache |
| Scale (1000+) | Shard by region | Multiple Supabase projects per region, API gateway layer |

---

*Dokumen ini adalah living document. Update setiap kali ada keputusan arsitektur baru.*
