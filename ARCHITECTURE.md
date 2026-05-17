# Arsitektur SaaS Omnichannel POS + E-commerce

> **Versi:** 1.1.0  
> **Tanggal:** 2026-05-17  
> **Status:** Draft — System Design

---

## Daftar Isi

1. [Ringkasan Arsitektur](#1-ringkasan-arsitektur)
2. [Strategi Multi-Tenancy](#2-strategi-multi-tenancy)
3. [Stack Teknologi](#3-stack-teknologi)
4. [Struktur Folder — Backend (Go)](#4-struktur-folder--backend-go)
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
│                      API GATEWAY (Go / Gin)                            │
│   JWT Middleware  │  Tenant Resolver  │  Rate Limiter  │  Logger       │
└─────────┬───────────────────┬──────────────────┬─────────────────────┘
          │                   │                  │
┌─────────▼──────┐  ┌─────────▼──────┐  ┌───────▼──────────┐
│  Core Handlers │  │  Channel Sync  │  │  Realtime Engine  │
│  - auth        │  │  - Shopee      │  │  - Supabase RT    │
│  - products    │  │  - Tokopedia   │  │  - Redis PubSub   │
│  - orders      │  │  - TikTok Shop │  │  - WebSocket      │
│  - inventory   │  │  - Website     │  └───────────────────┘
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

### Backend (Go)

| Komponen | Teknologi | Alasan |
|----------|-----------|--------|
| Language | **Go 1.22+** | Performa tinggi, concurrency native, binary tunggal |
| HTTP Framework | **Gin** | Routing cepat, middleware ecosystem matang |
| Database Driver | **pgx/v5** | PostgreSQL driver terbaik untuk Go, support named args |
| Query Builder | **sqlc** | Generate type-safe Go code dari SQL murni |
| Migration | **golang-migrate** | CLI migration dengan file SQL standar |
| Auth | **golang-jwt/jwt v5** | JWT parsing + validasi claim |
| Cache | **go-redis/v9** | Redis client resmi untuk Go |
| Job Queue | **Asynq** | Redis-based task queue, reliable, monitoring built-in |
| Realtime Bridge | **gorilla/websocket** | Push update stok ke klien |
| Config | **viper** | Load .env + YAML, hot reload |
| Logging | **slog** (stdlib) | Structured logging bawaan Go 1.21+ |
| Validation | **go-playground/validator** | Struct tag validation |
| API Docs | **swaggo/swag** | Generate OpenAPI 3.0 dari komentar |
| HTTP Client | **resty/v2** | Panggil API marketplace |
| Testing | **testify + pgxmock** | Unit & integration test |

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

## 4. Struktur Folder — Backend (Go)

Menggunakan **Clean Architecture** dengan pemisahan tegas antara domain, repository, service, dan handler.

```
backend/
├── cmd/
│   └── api/
│       └── main.go                    # Entry point: init config, DB, router, server
│
├── internal/                          # Kode privat aplikasi
│   │
│   ├── config/
│   │   └── config.go                  # Baca .env via viper, ekspos struct Config
│   │
│   ├── domain/                        # Business entities — pure Go structs, no dependencies
│   │   ├── tenant.go
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── order.go
│   │   └── inventory.go
│   │
│   ├── dto/                           # Request & Response structs (JSON binding)
│   │   ├── auth_dto.go
│   │   ├── tenant_dto.go
│   │   ├── product_dto.go
│   │   ├── order_dto.go
│   │   └── inventory_dto.go
│   │
│   ├── handler/                       # HTTP handlers (thin layer, hanya I/O)
│   │   ├── router.go                  # Daftarkan semua route + middleware
│   │   ├── auth_handler.go
│   │   ├── tenant_handler.go
│   │   ├── user_handler.go
│   │   ├── product_handler.go
│   │   ├── order_handler.go
│   │   ├── inventory_handler.go
│   │   └── webhook_handler.go         # Inbound webhook Shopee/Tokopedia/TikTok
│   │
│   ├── middleware/                    # Gin middleware
│   │   ├── auth.go                    # Verifikasi JWT, inject claims ke context
│   │   ├── tenant.go                  # Ekstrak & validasi tenant_id dari token
│   │   ├── rate_limit.go              # Per-tenant rate limiting via Redis
│   │   ├── cors.go
│   │   └── logger.go                  # Request logging dengan slog
│   │
│   ├── repository/                    # Data access layer
│   │   ├── sqlc/                      # Auto-generated oleh sqlc (JANGAN diedit manual)
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   ├── product.sql.go
│   │   │   ├── order.sql.go
│   │   │   └── inventory.sql.go
│   │   ├── interfaces.go              # Repository interface definitions
│   │   ├── tenant_repo.go
│   │   ├── user_repo.go
│   │   ├── product_repo.go
│   │   ├── order_repo.go
│   │   └── inventory_repo.go
│   │
│   ├── service/                       # Business logic layer
│   │   ├── auth_service.go
│   │   ├── tenant_service.go
│   │   ├── user_service.go
│   │   ├── product_service.go
│   │   ├── order_service.go
│   │   ├── inventory_service.go
│   │   └── sync_service.go            # Orkestrasi sinkronisasi stok
│   │
│   ├── channels/                      # Integrasi marketplace
│   │   ├── channel.go                 # Interface: UpdateStock, GetOrders, dll
│   │   ├── shopee/
│   │   │   ├── shopee.go              # Implementasi API Shopee
│   │   │   └── webhook.go             # Parse & validasi webhook Shopee
│   │   ├── tokopedia/
│   │   │   ├── tokopedia.go
│   │   │   └── webhook.go
│   │   └── tiktok/
│   │       ├── tiktok.go
│   │       └── webhook.go
│   │
│   └── worker/                        # Background jobs (Asynq)
│       ├── server.go                  # Inisialisasi Asynq server + mux
│       ├── tasks.go                   # Konstanta nama task
│       ├── stock_sync_handler.go      # Handle task broadcast stok ke marketplace
│       └── order_fulfill_handler.go   # Handle task fulfillment pesanan
│
├── pkg/                               # Paket reusable (bisa dipakai proyek lain)
│   ├── database/
│   │   └── postgres.go                # pgxpool connection dengan retry
│   ├── cache/
│   │   └── redis.go                   # Redis client wrapper
│   ├── jwt/
│   │   └── jwt.go                     # Parse, generate, validate JWT
│   ├── supabase/
│   │   └── client.go                  # Supabase REST client (auth, storage)
│   └── response/
│       └── response.go                # Helper: JSON success/error response
│
├── sql/
│   ├── schema/                        # DDL murni — source of truth skema
│   │   ├── 001_tenants.sql
│   │   ├── 002_users.sql
│   │   ├── 003_products.sql
│   │   ├── 004_orders.sql
│   │   └── 005_inventory_log.sql
│   └── queries/                       # Query SQL untuk sqlc
│       ├── product.sql
│       ├── order.sql
│       └── inventory.sql
│
├── migrations/                        # golang-migrate versioned files
│   ├── 000001_init_schema.up.sql
│   ├── 000001_init_schema.down.sql
│   ├── 000002_add_rls_policies.up.sql
│   └── 000002_add_rls_policies.down.sql
│
├── .env.example
├── sqlc.yaml                          # Konfigurasi sqlc codegen
├── Makefile                           # make run, make migrate, make sqlc, make test
├── Dockerfile
└── go.mod
```

### Contoh: `go.mod`

```
module github.com/ariesandjaya/omnichannel-backend

go 1.22

require (
    github.com/gin-gonic/gin              v1.10.0
    github.com/jackc/pgx/v5               v5.6.0
    github.com/golang-jwt/jwt/v5          v5.2.1
    github.com/redis/go-redis/v9          v9.5.3
    github.com/hibiken/asynq              v0.24.1
    github.com/gorilla/websocket          v1.5.3
    github.com/spf13/viper                v1.19.0
    github.com/go-playground/validator/v10 v10.22.0
    github.com/swaggo/swag                v1.16.3
    github.com/swaggo/gin-swagger         v1.6.0
    github.com/go-resty/resty/v2          v2.13.1
    github.com/golang-migrate/migrate/v4  v4.17.1
    github.com/stretchr/testify           v1.9.0
    github.com/google/uuid                v1.6.0
)
```

### Contoh: `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: postgresql
    queries: sql/queries
    schema: sql/schema
    gen:
      go:
        package: sqlcgen
        out: internal/repository/sqlc
        emit_json_tags: true
        emit_db_tags: true
        emit_prepared_queries: false
        emit_interface: true
```

### Contoh: `internal/handler/router.go`

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/ariesandjaya/omnichannel-backend/internal/middleware"
)

func NewRouter(
    auth     *AuthHandler,
    products *ProductHandler,
    orders   *OrderHandler,
    inventory *InventoryHandler,
    webhooks  *WebhookHandler,
    jwtSecret string,
) *gin.Engine {
    r := gin.New()
    r.Use(middleware.Logger(), middleware.CORS())

    // Public routes
    v1 := r.Group("/api/v1")
    v1.POST("/auth/login", auth.Login)
    v1.POST("/auth/register", auth.Register)

    // Webhook routes (autentikasi via HMAC signature, bukan JWT)
    webhookGroup := v1.Group("/webhooks")
    webhookGroup.POST("/shopee", webhooks.Shopee)
    webhookGroup.POST("/tokopedia", webhooks.Tokopedia)
    webhookGroup.POST("/tiktok", webhooks.TikTok)

    // Protected routes
    protected := v1.Group("/")
    protected.Use(middleware.JWTAuth(jwtSecret), middleware.ResolveTenant())
    {
        protected.GET("/products", products.List)
        protected.POST("/products", products.Create)
        protected.PUT("/products/:id", products.Update)
        protected.DELETE("/products/:id", products.Delete)

        protected.GET("/orders", orders.List)
        protected.POST("/orders", orders.Create)
        protected.PATCH("/orders/:id/status", orders.UpdateStatus)

        protected.GET("/inventory", inventory.List)
        protected.POST("/inventory/adjust", inventory.Adjust)
        protected.GET("/inventory/log", inventory.Log)
    }

    return r
}
```

### Contoh: `internal/middleware/auth.go`

```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    jwtlib "github.com/golang-jwt/jwt/v5"
)

type TenantClaims struct {
    TenantID string `json:"tenant_id"`
    Role     string `json:"role"`
    jwtlib.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        raw := c.GetHeader("Authorization")
        if !strings.HasPrefix(raw, "Bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }

        claims := &TenantClaims{}
        _, err := jwtlib.ParseWithClaims(
            strings.TrimPrefix(raw, "Bearer "),
            claims,
            func(t *jwtlib.Token) (any, error) { return []byte(secret), nil },
        )
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        c.Set("tenant_id", claims.TenantID)
        c.Set("user_id", claims.Subject)
        c.Set("role", claims.Role)
        c.Next()
    }
}
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
│       │   ├── page.tsx
│       │   ├── new/
│       │   ├── [id]/
│       │   └── categories/
│       │
│       ├── orders/
│       │   ├── page.tsx
│       │   └── [id]/
│       │
│       ├── inventory/
│       │   ├── page.tsx
│       │   ├── adjustments/
│       │   └── log/
│       │
│       ├── channels/
│       │   ├── page.tsx
│       │   ├── shopee/
│       │   ├── tokopedia/
│       │   └── website/
│       │
│       ├── reports/
│       │   ├── page.tsx
│       │   ├── sales/
│       │   └── inventory/
│       │
│       └── settings/
│           ├── page.tsx
│           ├── profile/
│           ├── team/
│           └── billing/
│
├── components/
│   ├── ui/                            # shadcn/ui base components
│   ├── pos/
│   │   ├── ProductGrid.tsx
│   │   ├── CartPanel.tsx
│   │   ├── PaymentModal.tsx
│   │   ├── ReceiptPrint.tsx
│   │   └── StockBadge.tsx
│   ├── products/
│   │   ├── ProductForm.tsx
│   │   ├── ProductTable.tsx
│   │   └── StockAlert.tsx
│   ├── inventory/
│   │   ├── InventoryLogTable.tsx
│   │   └── AdjustmentForm.tsx
│   ├── layout/
│   │   ├── Sidebar.tsx
│   │   ├── Header.tsx
│   │   └── MobileNav.tsx
│   └── shared/
│       ├── DataTable.tsx
│       ├── ConfirmDialog.tsx
│       ├── LoadingSpinner.tsx
│       └── EmptyState.tsx
│
├── hooks/
│   ├── useRealtimeStock.ts
│   ├── useTenant.ts
│   ├── usePermissions.ts
│   └── useOfflineCart.ts
│
├── lib/
│   ├── supabase/
│   │   ├── client.ts
│   │   ├── server.ts
│   │   └── middleware.ts
│   ├── api/
│   │   ├── products.ts
│   │   ├── orders.ts
│   │   └── inventory.ts
│   └── utils/
│       ├── currency.ts
│       ├── date.ts
│       └── validators.ts
│
├── stores/
│   ├── cart.store.ts
│   ├── tenant.store.ts
│   └── ui.store.ts
│
├── types/
│   ├── database.ts
│   ├── api.ts
│   └── pos.ts
│
├── middleware.ts
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
    -- {"shopee": "123456", "tokopedia": "789"}
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
    -- pos | website | shopee | tokopedia | tiktok_shop | manual
  channel_order_id VARCHAR(255),
  status           VARCHAR(50)   NOT NULL DEFAULT 'pending',
    -- pending | confirmed | processing | shipped | completed | cancelled | refunded
  customer_id      UUID          REFERENCES users(id),
  customer_info    JSONB         NOT NULL DEFAULT '{}',
  line_items       JSONB         NOT NULL DEFAULT '[]',
    -- [{product_id, sku, name, quantity, unit_price, discount, subtotal, snapshot}]
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
    -- sale | purchase | adjustment | transfer_in | transfer_out | return | correction
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
```

---

## 7. Row-Level Security (RLS)

RLS memastikan bahwa **setiap query** otomatis difilter hanya untuk `tenant_id` yang sesuai dengan JWT token, sebagai lapisan keamanan kedua di level database.

```sql
-- Aktifkan RLS pada semua tabel
ALTER TABLE tenants            ENABLE ROW LEVEL SECURITY;
ALTER TABLE users              ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE products           ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders             ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_log      ENABLE ROW LEVEL SECURITY;


-- Helper: ekstrak tenant_id dari JWT Supabase
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
  SELECT COALESCE(
    (current_setting('request.jwt.claims', true)::jsonb ->> 'tenant_id')::UUID,
    NULL
  );
$$ LANGUAGE SQL STABLE;


-- Isolasi per-tenant untuk semua tabel
CREATE POLICY users_tenant_isolation ON users
  FOR ALL USING (tenant_id = current_tenant_id());

CREATE POLICY products_tenant_isolation ON products
  FOR ALL USING (tenant_id = current_tenant_id());

-- Storefront publik: produk aktif bisa dibaca tanpa auth
CREATE POLICY products_public_read ON products
  FOR SELECT USING (
    is_active = TRUE
    AND 'website' = ANY(SELECT jsonb_array_elements_text(channels))
  );

CREATE POLICY orders_tenant_isolation ON orders
  FOR ALL USING (tenant_id = current_tenant_id());

CREATE POLICY inventory_log_tenant_isolation ON inventory_log
  FOR ALL USING (tenant_id = current_tenant_id());

-- Log immutable: tolak UPDATE dan DELETE
CREATE POLICY inventory_log_no_update ON inventory_log FOR UPDATE USING (FALSE);
CREATE POLICY inventory_log_no_delete ON inventory_log FOR DELETE USING (FALSE);
```

---

## 8. Sinkronisasi Stok Real-time

### Arsitektur Sync

```
[Shopee Webhook]
       │ POST /api/v1/webhooks/shopee
       ▼
[webhook_handler.go]            ← validasi HMAC signature
       │
       ▼
[order_service.go]
       │ 1. Simpan order ke DB
       │ 2. Panggil deduct_stock() — atomic SQL function
       │ 3. Enqueue Asynq task
       ▼
[PostgreSQL]
       │ products.stock_quantity berubah
       │ → Supabase Realtime broadcast otomatis
       ▼
[Supabase Realtime]
       ├──► POS Terminal (JS client subscribe)
       └──► Dashboard Web (JS client subscribe)

[Asynq Worker — stock_sync_handler.go]
       │ Goroutine per channel (concurrent)
       ├──► Shopee API: update_stock
       ├──► Tokopedia API: update_stock
       └──► TikTok Shop API: update_stock
```

### Atomic Stock Deduction (PostgreSQL Function)

```sql
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
    tenant_id, product_id, type,
    quantity_change, quantity_before, quantity_after,
    reference_type, reference_id, channel, created_by
  ) VALUES (
    p_tenant_id, p_product_id, 'sale',
    -p_quantity, v_before, v_after,
    'order', p_reference_id, p_channel, p_user_id
  );

  RETURN QUERY SELECT TRUE, 'OK'::TEXT, v_after;
END;
$$;
```

### Asynq Worker — Go

```go
// internal/worker/stock_sync_handler.go
package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"

    "github.com/hibiken/asynq"
)

const TaskBroadcastStock = "stock:broadcast"

type StockSyncPayload struct {
    TenantID    string   `json:"tenant_id"`
    ProductID   string   `json:"product_id"`
    NewQuantity int      `json:"new_quantity"`
    Channels    []string `json:"channels"`
}

func (h *SyncHandler) HandleBroadcastStock(ctx context.Context, t *asynq.Task) error {
    var p StockSyncPayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil {
        return fmt.Errorf("unmarshal: %w", err)
    }

    var wg sync.WaitGroup
    errs := make(chan error, len(p.Channels))

    for _, ch := range p.Channels {
        wg.Add(1)
        go func(channel string) {
            defer wg.Done()
            var err error
            switch channel {
            case "shopee":
                err = h.shopee.UpdateStock(ctx, p.TenantID, p.ProductID, p.NewQuantity)
            case "tokopedia":
                err = h.tokopedia.UpdateStock(ctx, p.TenantID, p.ProductID, p.NewQuantity)
            case "tiktok":
                err = h.tiktok.UpdateStock(ctx, p.TenantID, p.ProductID, p.NewQuantity)
            }
            if err != nil {
                errs <- fmt.Errorf("%s: %w", channel, err)
            }
        }(ch)
    }

    wg.Wait()
    close(errs)

    for err := range errs {
        h.logger.Error("channel sync failed", "error", err)
    }
    return nil
}
```

### Supabase Realtime — Subscribe di Frontend

```typescript
// hooks/useRealtimeStock.ts
import { useEffect } from 'react'
import { supabase } from '@/lib/supabase/client'
import { useCartStore } from '@/stores/cart.store'

export function useRealtimeStock(tenantId: string) {
  const updateStock = useCartStore(s => s.updateProductStock)

  useEffect(() => {
    const channel = supabase
      .channel(`stock:${tenantId}`)
      .on('postgres_changes', {
        event: 'UPDATE',
        schema: 'public',
        table: 'products',
        filter: `tenant_id=eq.${tenantId}`,
      }, payload => {
        updateStock(payload.new.id, payload.new.stock_quantity)
      })
      .subscribe()

    return () => { supabase.removeChannel(channel) }
  }, [tenantId])
}
```

---

## 9. Alur Autentikasi & Otorisasi

```
1. User login  →  Supabase Auth (email/password / OAuth)
2. Supabase mengembalikan JWT yang sudah di-embed tenant_id + role
3. Setiap request API menyertakan: Authorization: Bearer <token>
4. Go middleware (internal/middleware/auth.go) parse JWT → ekstrak tenant_id
5. middleware/tenant.go validasi tenant aktif, inject ke gin.Context
6. Service layer pakai tenant_id dari context di setiap query
7. PostgreSQL RLS memvalidasi tenant_id sebagai safety net ke-2
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

### Channel Interface (Go)

Semua integrasi marketplace mengimplementasikan interface yang sama:

```go
// internal/channels/channel.go
package channels

import "context"

type MarketplaceChannel interface {
    UpdateStock(ctx context.Context, tenantID, productID string, qty int) error
    GetNewOrders(ctx context.Context, tenantID string) ([]ExternalOrder, error)
    ConfirmOrder(ctx context.Context, tenantID, channelOrderID string) error
    ValidateWebhookSignature(payload []byte, signature string) bool
}
```

---

## 10. Deployment & Infrastruktur

```
┌──────────────────────────────────────────────────────────────┐
│                      Production Stack                         │
├──────────────────┬──────────────────────┬────────────────────┤
│   Frontend       │    Backend           │    Data Layer      │
│   Vercel         │    Fly.io / Railway  │    Supabase (PaaS) │
│   (Next.js)      │    (Go binary)       │    PostgreSQL +    │
│                  │                      │    Realtime +      │
│                  │    Redis (Upstash)   │    Auth + Storage  │
│                  │    Asynq Workers     │                    │
└──────────────────┴──────────────────────┴────────────────────┘
```

### Keunggulan Go untuk Deployment

| Aspek | NestJS | Go |
|-------|--------|----|
| Binary size | ~300MB (node_modules) | ~15MB (single binary) |
| Cold start | ~3–5 detik | < 100ms |
| Memory usage | ~150MB idle | ~20MB idle |
| Concurrency | Event loop | Native goroutines |
| Docker image | ~500MB | ~20MB (scratch/alpine) |

### Dockerfile (Multi-stage)

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# Stage 2: Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
```

### Environment Variables (Backend)

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

### Makefile

```makefile
run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:
	go test ./... -v -race

migrate-up:
	golang-migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	golang-migrate -path migrations -database "$(DATABASE_URL)" down 1

sqlc:
	sqlc generate

swag:
	swag init -g cmd/api/main.go

docker-build:
	docker build -t omnichannel-api:latest .
```

### Skalabilitas

| Fase | Tenant | Strategi |
|------|--------|----------|
| MVP (0–100) | 1 instance | Fly.io single machine, 1 Supabase project |
| Growth (100–1000) | Scale horizontal | 2–3 Go instances, Redis untuk shared state, PgBouncer |
| Scale (1000+) | Shard by region | Multi-region Fly.io, read replica Supabase, CDN untuk aset |

---

*Dokumen ini adalah living document. Update setiap kali ada keputusan arsitektur baru.*
