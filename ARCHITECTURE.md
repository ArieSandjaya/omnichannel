# Arsitektur SaaS Omnichannel POS + E-commerce

> **Versi:** 1.2.0  
> **Tanggal:** 2026-05-17  
> **Status:** Draft — System Design

---

## Daftar Isi

1. [Ringkasan Arsitektur](#1-ringkasan-arsitektur)
2. [Strategi Multi-Tenancy](#2-strategi-multi-tenancy)
3. [Stack Teknologi](#3-stack-teknologi)
4. [Struktur Folder — Backend (Go API)](#4-struktur-folder--backend-go-api)
5. [Struktur Folder — Frontend (Go + Templ + HTMX)](#5-struktur-folder--frontend-go--templ--htmx)
6. [Skema Database](#6-skema-database)
7. [Row-Level Security (RLS)](#7-row-level-security-rls)
8. [Sinkronisasi Stok Real-time](#8-sinkronisasi-stok-real-time)
9. [Alur Autentikasi & Otorisasi](#9-alur-autentikasi--otorisasi)
10. [Deployment & Infrastruktur](#10-deployment--infrastruktur)

---

## 1. Ringkasan Arsitektur

Platform ini adalah **SaaS multi-tenant full-stack Go** tanpa ketergantungan pada Node.js sama sekali. Setiap tenant memiliki:

- Manajemen produk & stok terpusat
- Titik penjualan (POS) di toko fisik
- Storefront e-commerce online
- Integrasi marketplace (Shopee, Tokopedia, TikTok Shop)
- Sinkronisasi stok real-time lintas channel

```
┌───────────────────────────────────────────────────────────────┐
│                      BROWSER                                   │
│   HTMX + Alpine.js + Tailwind (served oleh Go, tanpa Node.js)  │
└───────────────────────┬────────────────┬──────────────────┘
                       │ HTTPS              │ SSE (stock updates)
┌───────────────────────▼────────────────▼──────────────────┐
│                GO WEB SERVER (Gin)                             │
│  ┌────────────────────┐  ┌────────────────────┐  │
│  │  Page Handlers    │  │   REST API           │  │
│  │  (render Templ)   │  │   /api/v1/...        │  │
│  └────────────────────┘  └────────────────────┘  │
└───────────────────────┬────────────────────────────────────┘
                       │
          ┌──────────────────────────────────┐
          │        Service Layer            │
          │  products | orders | inventory  │
          │  channels | sync | auth         │
          └────────────┬─────────────────────┘
                       │
┌───────────────────────▼────────────────────────────────────┐
│               PostgreSQL via Supabase (RLS)                    │
└────────────────────────────────────────────────────────────┘
```

> **Catatan Arsitektur:** Backend (REST API) dan Frontend (page rendering) dijalankan sebagai **satu binary Go tunggal**. Page handler me-render Templ templates, sementara HTMX di browser melakukan request HTTPS ke REST API untuk operasi data. SSE digunakan untuk push update stok real-time dari server ke browser.

---

## 2. Strategi Multi-Tenancy

### Pendekatan: Shared Database + Shared Schema + Row-Level Security

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

> **Prinsip:** Zero Node.js. Semua komponen berjalan tanpa runtime Node.js atau npm/yarn.

### Backend (Go — REST API)

| Komponen | Teknologi | Alasan |
|----------|-----------|--------|
| Language | **Go 1.22+** | Performa tinggi, concurrency native, binary tunggal |
| HTTP Framework | **Gin** | Routing cepat, middleware ecosystem matang |
| Database Driver | **pgx/v5** | PostgreSQL driver terbaik untuk Go |
| Query Builder | **sqlc** | Generate type-safe Go code dari SQL murni |
| Migration | **golang-migrate** | CLI migration dengan file SQL standar |
| Auth | **golang-jwt/jwt v5** | JWT parsing + validasi claim |
| Cache | **go-redis/v9** | Redis client resmi untuk Go |
| Job Queue | **Asynq** | Redis-based task queue, monitoring built-in |
| Config | **viper** | Load .env + YAML |
| Logging | **slog** (stdlib) | Structured logging bawaan Go 1.21+ |
| Validation | **go-playground/validator** | Struct tag validation |
| API Docs | **swaggo/swag** | Generate OpenAPI 3.0 dari komentar |
| HTTP Client | **resty/v2** | Panggil API marketplace |
| Testing | **testify + pgxmock** | Unit & integration test |

### Frontend (Go — Server-Side Rendered)

| Komponen | Teknologi | Alasan |
|----------|-----------|--------|
| Templating | **Templ** | Type-safe HTML templates, dikompilasi ke Go code |
| Interaktivitas | **HTMX** (via CDN / file lokal) | Dynamic UI tanpa JS framework, tidak perlu build |
| Micro-UI | **Alpine.js** (via CDN / file lokal) | Dropdown, modal, form state — deklaratif, minimal |
| Styling | **Tailwind CSS** (standalone CLI binary) | Utility CSS tanpa npm, binary tunggal |
| Realtime | **SSE** (Server-Sent Events via Go) | Push stok ke browser, built-in di HTTP standard |
| POS Offline | **IndexedDB** via vanilla JS | Cart persistence saat koneksi terputus |
| Icon | **Heroicons** (inline SVG) | Tidak butuh icon font atau build step |

**Mengapa tidak perlu Node.js:**
- Templ di-compile ke Go code → tidak butuh bundler JS
- HTMX & Alpine.js di-download sekali sebagai file statis
- Tailwind CSS menggunakan [standalone CLI](https://tailwindcss.com/blog/standalone-cli) — binary Go/Rust, bukan npm package
- SSE menggantikan WebSocket/Supabase JS client untuk realtime

---

## 4. Struktur Folder — Backend (Go API)

```
omnichannel/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point: init semua deps, jalankan Gin
│
├── internal/
│   ├── config/
│   │   └── config.go
│   │
│   ├── domain/                        # Pure Go structs, tidak ada import eksternal
│   │   ├── tenant.go
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── order.go
│   │   └── inventory.go
│   │
│   ├── dto/
│   │   ├── auth_dto.go
│   │   ├── product_dto.go
│   │   ├── order_dto.go
│   │   └── inventory_dto.go
│   │
│   ├── handler/
│   │   ├── router.go                  # Daftarkan semua route
│   │   ├── auth_handler.go            # POST /api/v1/auth/*
│   │   ├── product_handler.go         # CRUD /api/v1/products
│   │   ├── order_handler.go
│   │   ├── inventory_handler.go
│   │   ├── webhook_handler.go         # POST /webhooks/shopee|tokopedia|tiktok
│   │   ├── sse_handler.go             # GET /sse/stock — Server-Sent Events
│   │   └── page_handler.go            # GET /pos, /products, /orders, dll (render Templ)
│   │
│   ├── middleware/
│   │   ├── auth.go                    # Verifikasi JWT
│   │   ├── tenant.go                  # Inject tenant_id ke context
│   │   ├── rate_limit.go
│   │   ├── cors.go
│   │   └── logger.go
│   │
│   ├── repository/
│   │   ├── sqlc/                      # Auto-generated oleh sqlc
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   ├── product.sql.go
│   │   │   ├── order.sql.go
│   │   │   └── inventory.sql.go
│   │   ├── interfaces.go
│   │   ├── product_repo.go
│   │   ├── order_repo.go
│   │   └── inventory_repo.go
│   │
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── product_service.go
│   │   ├── order_service.go
│   │   ├── inventory_service.go
│   │   └── sync_service.go
│   │
│   ├── broker/
│   │   └── sse_broker.go              # In-memory pub/sub untuk SSE per tenant
│   │
│   ├── channels/
│   │   ├── channel.go                 # Interface MarketplaceChannel
│   │   ├── shopee/
│   │   ├── tokopedia/
│   │   └── tiktok/
│   │
│   └── worker/
│       ├── server.go
│       ├── tasks.go
│       ├── stock_sync_handler.go
│       └── order_fulfill_handler.go
│
├── pkg/
│   ├── database/postgres.go
│   ├── cache/redis.go
│   ├── jwt/jwt.go
│   └── response/response.go
│
├── sql/
│   ├── schema/
│   └── queries/
│
├── migrations/
├── templates/                         # Templ files (lihat Section 5)
├── static/                            # Asset statis (lihat Section 5)
├── .env.example
├── sqlc.yaml
├── Makefile
├── Dockerfile
└── go.mod
```

---

## 5. Struktur Folder — Frontend (Go + Templ + HTMX)

Frontend **tidak berdiri sendiri** — ia adalah bagian dari binary Go yang sama. Templ di-compile menjadi Go code, lalu di-serve oleh Gin.

### Cara Kerja

```
[.templ files]
     │
     │  templ generate  (saat development / CI)
     ▼
[*_templ.go files]  ──►  dikompilasi bersama go build
     │
     ▼
[page_handler.go memanggil template]
     │
     ▼
[Browser menerima HTML + HTMX attributes]
     │
     │  User interaksi (klik, submit form)
     ▼
[HTMX kirim request ke /api/v1/... atau ke page handler untuk partial]
     │
     ▼
[Server balas dengan HTML fragment atau JSON]
     │
     ▼
[HTMX swap fragment ke DOM — tanpa reload halaman]
```

### Struktur Folder

```
omnichannel/
├── templates/
│   ├── layouts/
│   │   ├── base.templ                 # HTML shell: head, script, CSS link
│   │   ├── dashboard.templ            # Layout dengan sidebar + header
│   │   └── auth.templ                 # Layout halaman login/register
│   │
│   ├── pages/                         # Full-page templates
│   │   ├── login.templ
│   │   ├── register.templ
│   │   ├── dashboard.templ            # Overview / home
│   │   ├── pos.templ                  # POS terminal
│   │   ├── products.templ             # Daftar produk
│   │   ├── product_form.templ         # Create/edit produk
│   │   ├── orders.templ
│   │   ├── order_detail.templ
│   │   ├── inventory.templ
│   │   ├── channels.templ
│   │   ├── reports.templ
│   │   └── settings.templ
│   │
│   ├── components/                    # Partial / reusable template components
│   │   ├── sidebar.templ
│   │   ├── header.templ
│   │   ├── data_table.templ           # Generic table dengan pagination
│   │   ├── confirm_dialog.templ
│   │   ├── flash_message.templ        # Toast/alert notifikasi
│   │   ├── pagination.templ
│   │   ├── pos/
│   │   │   ├── product_grid.templ     # Grid produk untuk kasir
│   │   │   ├── cart_panel.templ       # Panel keranjang
│   │   │   ├── payment_modal.templ    # Modal proses bayar
│   │   │   ├── receipt.templ          # Struk cetak
│   │   │   └── stock_badge.templ      # Badge stok (di-swap via SSE + HTMX)
│   │   ├── products/
│   │   │   ├── product_row.templ      # Baris tabel produk (HTMX swap)
│   │   │   └── stock_alert.templ      # Alert stok menipis
│   │   └── inventory/
│   │       ├── log_table.templ
│   │       └── adjustment_form.templ
│   │
│   └── partials/                      # HTML fragments untuk HTMX swap
│       ├── product_search_results.templ  # Hasil search produk di POS
│       ├── cart_items.templ           # Update keranjang
│       ├── order_status_badge.templ   # Badge status pesanan
│       └── stock_quantity.templ       # Angka stok (di-replace via SSE)
│
└── static/
    ├── css/
    │   ├── input.css                  # Tailwind directives (@tailwind base, dll)
    │   └── app.css                    # Output Tailwind (dihasilkan oleh standalone CLI)
    ├── js/
    │   ├── htmx.min.js                # Download dari unpkg, tidak perlu npm
    │   ├── alpine.min.js              # Download dari unpkg, tidak perlu npm
    │   └── pos-offline.js             # Vanilla JS: IndexedDB cart persistence
    ├── img/
    │   └── logo.svg
    └── favicon.ico
```

### Contoh: `templates/layouts/base.templ`

```go
package layouts

templ Base(title string) {
    <!DOCTYPE html>
    <html lang="id">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{ title } — Omnichannel POS</title>
        <link rel="stylesheet" href="/static/css/app.css"/>
        <script src="/static/js/htmx.min.js" defer></script>
        <script src="/static/js/alpine.min.js" defer></script>
    </head>
    <body class="bg-gray-50 text-gray-900">
        { children... }
    </body>
    </html>
}
```

### Contoh: `templates/components/pos/stock_badge.templ`

```go
package pos

templ StockBadge(productID string, qty int) {
    <span
        id={ "stock-" + productID }
        class={ badgeClass(qty) }>
        { strconv.Itoa(qty) } pcs
    </span>
}

func badgeClass(qty int) string {
    if qty == 0 {
        return "badge badge-error"
    }
    if qty <= 5 {
        return "badge badge-warning"
    }
    return "badge badge-success"
}
```

### Contoh: HTMX pada halaman POS

```html
<!-- Search produk — kirim request saat user mengetik, swap hasilnya -->
<input
  type="search"
  name="q"
  placeholder="Cari produk atau scan barcode..."
  hx-get="/partials/products/search"
  hx-trigger="keyup changed delay:300ms"
  hx-target="#product-grid"
  hx-swap="innerHTML"
  class="w-full px-4 py-2 border rounded-lg"
/>

<!-- Tambah item ke keranjang -->
<button
  hx-post="/partials/cart/add"
  hx-vals='{"product_id": "{{ .ID }}"}'
  hx-target="#cart-panel"
  hx-swap="innerHTML"
  class="btn btn-primary w-full">
  + Tambah
</button>

<!-- Subscribe SSE untuk update stok real-time -->
<div
  hx-ext="sse"
  sse-connect="/sse/stock"
  sse-swap="stock-update"
  hx-swap="none"
  id="sse-stock-listener">
</div>
```

### Alpine.js — State UI Lokal

```html
<!-- Modal pembayaran dengan Alpine.js -->
<div x-data="{ open: false, method: 'cash', amount: 0 }">
    <button @click="open = true" class="btn btn-success">
        Proses Pembayaran
    </button>

    <div x-show="open" x-cloak class="modal-overlay">
        <div class="modal">
            <select x-model="method">
                <option value="cash">Tunai</option>
                <option value="qris">QRIS</option>
                <option value="card">Kartu</option>
            </select>

            <input x-show="method === 'cash'"
                   x-model.number="amount"
                   type="number"
                   placeholder="Nominal diterima"/>

            <p x-show="method === 'cash' && amount > 0">
                Kembalian: <span x-text="amount - total"></span>
            </p>

            <!-- HTMX submit form pembayaran -->
            <button
                hx-post="/api/v1/orders"
                hx-include="#cart-form"
                @click="open = false"
                class="btn btn-primary">
                Bayar
            </button>
        </div>
    </div>
</div>
```

### Tailwind CSS — Tanpa npm

```makefile
# Download Tailwind standalone binary (sekali saja)
install-tailwind:
    curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
    chmod +x tailwindcss-linux-x64
    mv tailwindcss-linux-x64 ./bin/tailwindcss

# Generate CSS (tidak butuh npm)
css:
    ./bin/tailwindcss -i static/css/input.css -o static/css/app.css --minify

# Watch mode saat development
css-watch:
    ./bin/tailwindcss -i static/css/input.css -o static/css/app.css --watch
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
    -- super_admin | owner | manager | cashier | staff | viewer
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
```

---

## 7. Row-Level Security (RLS)

```sql
ALTER TABLE tenants            ENABLE ROW LEVEL SECURITY;
ALTER TABLE users              ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE products           ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders             ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_log      ENABLE ROW LEVEL SECURITY;


CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
  SELECT COALESCE(
    (current_setting('request.jwt.claims', true)::jsonb ->> 'tenant_id')::UUID,
    NULL
  );
$$ LANGUAGE SQL STABLE;


CREATE POLICY users_tenant_isolation ON users
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

CREATE POLICY inventory_log_no_update ON inventory_log FOR UPDATE USING (FALSE);
CREATE POLICY inventory_log_no_delete ON inventory_log FOR DELETE USING (FALSE);
```

---

## 8. Sinkronisasi Stok Real-time

### Arsitektur Sync

```
[Webhook / POS Order]
       │ POST /api/v1/orders
       ▼
[order_service.go]
       │ 1. Simpan order ke DB
       │ 2. Panggil deduct_stock() — atomic SQL
       │ 3. Publish ke SSE broker (in-memory per tenant)
       │ 4. Enqueue Asynq task: sync-to-marketplaces
       ▼
[SSE Broker — sse_broker.go]
       │ Broadcast StockUpdateEvent ke semua
       │ koneksi SSE milik tenant tersebut
       ▼
[Browser — HTMX SSE extension]
       │ Menerima event "stock-update"
       │ HTMX swap fragment #stock-{productID}
       ▼
[stock_quantity.templ di-render ulang]
  (angka stok terupdate tanpa reload)

[Asynq Worker]
       ├──► Shopee API: update_stock
       ├──► Tokopedia API: update_stock
       └──► TikTok Shop API: update_stock
```

### SSE Broker — Go

```go
// internal/broker/sse_broker.go
package broker

import "sync"

type StockEvent struct {
    ProductID string `json:"product_id"`
    Quantity  int    `json:"quantity"`
    TenantID  string `json:"tenant_id"`
}

type SSEBroker struct {
    mu          sync.RWMutex
    subscribers map[string][]chan StockEvent // key: tenant_id
}

func NewSSEBroker() *SSEBroker {
    return &SSEBroker{subscribers: make(map[string][]chan StockEvent)}
}

func (b *SSEBroker) Subscribe(tenantID string) chan StockEvent {
    ch := make(chan StockEvent, 10)
    b.mu.Lock()
    b.subscribers[tenantID] = append(b.subscribers[tenantID], ch)
    b.mu.Unlock()
    return ch
}

func (b *SSEBroker) Unsubscribe(tenantID string, ch chan StockEvent) {
    b.mu.Lock()
    defer b.mu.Unlock()
    subs := b.subscribers[tenantID]
    for i, s := range subs {
        if s == ch {
            b.subscribers[tenantID] = append(subs[:i], subs[i+1:]...)
            close(ch)
            return
        }
    }
}

func (b *SSEBroker) Publish(event StockEvent) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers[event.TenantID] {
        select {
        case ch <- event:
        default: // skip jika subscriber lambat
        }
    }
}
```

### SSE Handler — Go

```go
// internal/handler/sse_handler.go
package handler

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/ariesandjaya/omnichannel/internal/broker"
)

type SSEHandler struct {
    broker *broker.SSEBroker
}

func (h *SSEHandler) StreamStock(c *gin.Context) {
    tenantID := c.GetString("tenant_id")

    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

    ch := h.broker.Subscribe(tenantID)
    defer h.broker.Unsubscribe(tenantID, ch)

    for {
        select {
        case event := <-ch:
            data, _ := json.Marshal(event)
            fmt.Fprintf(c.Writer, "event: stock-update\ndata: %s\n\n", data)
            c.Writer.Flush()
        case <-c.Request.Context().Done():
            return
        }
    }
}
```

### HTMX di Browser — Terima SSE

```html
<!-- Pasang listener SSE, HTMX akan auto-request partial saat ada event -->
<div
  hx-ext="sse"
  sse-connect="/sse/stock"
  id="sse-listener">

  <!-- Saat event "stock-update" datang, request partial dan swap -->
  <div
    hx-get="/partials/stock-quantities"
    sse-trigger="stock-update"
    hx-target="#product-grid"
    hx-swap="none"
    hx-select="[data-stock-badge]"
    hx-swap-oob="true">
  </div>
</div>
```

### Atomic Stock Deduction (PostgreSQL Function)

```sql
CREATE OR REPLACE FUNCTION deduct_stock(
  p_tenant_id UUID, p_product_id UUID, p_quantity INTEGER,
  p_reference_id UUID, p_channel VARCHAR(50), p_user_id UUID
)
RETURNS TABLE(success BOOLEAN, message TEXT, new_quantity INTEGER)
LANGUAGE plpgsql AS $$
DECLARE v_before INTEGER; v_after INTEGER;
BEGIN
  SELECT stock_quantity INTO v_before
  FROM products
  WHERE id = p_product_id AND tenant_id = p_tenant_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RETURN QUERY SELECT FALSE, 'Produk tidak ditemukan'::TEXT, 0; RETURN;
  END IF;

  IF v_before < p_quantity THEN
    RETURN QUERY SELECT FALSE, 'Stok tidak mencukupi'::TEXT, v_before; RETURN;
  END IF;

  v_after := v_before - p_quantity;

  UPDATE products SET stock_quantity = v_after, updated_at = NOW()
  WHERE id = p_product_id AND tenant_id = p_tenant_id;

  INSERT INTO inventory_log (
    tenant_id, product_id, type, quantity_change,
    quantity_before, quantity_after, reference_type,
    reference_id, channel, created_by
  ) VALUES (
    p_tenant_id, p_product_id, 'sale', -p_quantity,
    v_before, v_after, 'order', p_reference_id, p_channel, p_user_id
  );

  RETURN QUERY SELECT TRUE, 'OK'::TEXT, v_after;
END;
$$;
```

---

## 9. Alur Autentikasi & Otorisasi

```
1. User buka /login  →  page_handler.go render login.templ
2. Submit form  →  POST /api/v1/auth/login
3. auth_service.go verifikasi ke Supabase Auth
4. Server set cookie HttpOnly: jwt=<token>
5. Redirect ke /dashboard
6. Setiap request berikutnya: middleware/auth.go baca cookie
7. Parse JWT → ekstrak tenant_id + role ke gin.Context
8. Service layer pakai tenant_id dari context
9. PostgreSQL RLS sebagai safety net ke-2
```

### Cookie vs Authorization Header

Karena frontend adalah server-rendered (bukan SPA), **HttpOnly cookie** lebih aman dari localStorage:

```go
// internal/handler/auth_handler.go
func (h *AuthHandler) Login(c *gin.Context) {
    // ... validasi kredensial ...

    c.SetCookie(
        "jwt",       // name
        token,       // value
        60*60*24*7,  // maxAge: 7 hari
        "/",         // path
        "",          // domain
        true,        // secure (HTTPS only)
        true,        // httpOnly (tidak bisa diakses JS)
    )
    c.Redirect(http.StatusFound, "/dashboard")
}
```

### Role Hierarchy

```
super_admin   → Akses semua tenant (platform admin)
    └── owner         → Akses penuh satu tenant
          └── manager       → Operasional (tidak bisa billing)
                └── cashier       → Hanya POS
                      └── staff         → Akses terbatas
                            └── viewer        → Read-only
```

---

## 10. Deployment & Infrastruktur

```
┌──────────────────────────────────────────────────────────────┐
│                      Production Stack                         │
├──────────────────┬───────────────────────┬───────────────────┤
│   Go Binary (single) │    Job Worker         │    Data Layer      │
│   Fly.io / Railway   │    (Asynq)            │    Supabase (PaaS) │
│                      │    Fly.io             │    PostgreSQL +    │
│   Serve HTML pages   │                       │    Auth + Storage  │
│   + REST API         │    Redis (Upstash)    │                    │
│   + SSE stream       │                       │                    │
└──────────────────┴───────────────────────┴───────────────────┘
```

### Keunggulan Full-Stack Go (Zero Node.js)

| Aspek | Node.js (Next.js) | Go + Templ + HTMX |
|-------|-------------------|--------------------|
| Runtime dependency | Node.js v20+ | Tidak ada |
| Cold start | ~3–5 detik | < 100ms |
| Memory idle | ~150MB | ~20MB |
| Build step frontend | npm install + webpack | `templ generate` + `go build` |
| Docker image | ~500MB | ~20MB |
| Deploy | Vercel + API server | 1 binary, 1 server |
| JS di browser | React (heavy) | HTMX + Alpine.js (~25KB total) |

### Dockerfile (Single Binary)

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Download Tailwind binary
RUN wget -q https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    -O /usr/local/bin/tailwindcss && chmod +x /usr/local/bin/tailwindcss

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate templates + CSS + binary
RUN templ generate
RUN tailwindcss -i static/css/input.css -o static/css/app.css --minify
RUN CGO_ENABLED=0 GOOS=linux go build -o /omnichannel ./cmd/server


FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /omnichannel /omnichannel
COPY --from=builder /app/static /static
EXPOSE 8080
ENTRYPOINT ["/omnichannel"]
```

### Makefile Lengkap

```makefile
.PHONY: run build templ css migrate sqlc test

run: templ css
	go run ./cmd/server

build: templ css
	CGO_ENABLED=0 go build -o bin/omnichannel ./cmd/server

templ:
	templ generate

css:
	./bin/tailwindcss -i static/css/input.css -o static/css/app.css --minify

css-watch:
	./bin/tailwindcss -i static/css/input.css -o static/css/app.css --watch

migrate-up:
	golang-migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	golang-migrate -path migrations -database "$(DATABASE_URL)" down 1

sqlc:
	sqlc generate

test:
	go test ./... -v -race

docker-build:
	docker build -t omnichannel:latest .

install-tools:
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/hibiken/asynq/tools/asynq@latest
	wget -q https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
	    -O bin/tailwindcss && chmod +x bin/tailwindcss
```

### Environment Variables

```env
# App
APP_ENV=production
APP_PORT=8080

# Database
DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=require

# Supabase
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...

# Redis
REDIS_URL=rediss://:token@host:6380

# JWT
JWT_SECRET=...
JWT_EXPIRES_IN=168h

# Marketplace
SHOPEE_PARTNER_ID=...
SHOPEE_PARTNER_KEY=...
TOKOPEDIA_CLIENT_ID=...
TOKOPEDIA_CLIENT_SECRET=...
TIKTOK_APP_KEY=...
TIKTOK_APP_SECRET=...
```

### Skalabilitas

| Fase | Tenant | Strategi |
|------|--------|----------|
| MVP (0–100) | 1 instance | Fly.io single machine 256MB RAM |
| Growth (100–1000) | 2–3 instance | Load balancer, Redis untuk SSE broker terdistribusi |
| Scale (1000+) | Horizontal scale | Redis PubSub untuk SSE lintas instance, read replica DB |

> **Catatan SSE multi-instance:** Saat scaling ke beberapa instance, `SSEBroker` in-memory harus diganti dengan **Redis PubSub** agar event dari instance A diterima oleh klien di instance B.

---

*Dokumen ini adalah living document. Update setiap kali ada keputusan arsitektur baru.*
