-- name: GetProductByID :one
SELECT id, tenant_id, category_id, sku, barcode, name, description,
       price, cost_price, compare_at_price, stock_quantity, min_stock_level,
       track_inventory, unit, weight, images, attributes, channels,
       channel_listing, is_active, created_by, created_at, updated_at
FROM products
WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE;

-- name: GetProductForUpdate :one
-- Locks the row with SELECT FOR UPDATE inside an active transaction.
-- Concurrent transactions requesting the same product will block until commit/rollback.
SELECT id, tenant_id, category_id, sku, barcode, name, description,
       price, cost_price, compare_at_price, stock_quantity, min_stock_level,
       track_inventory, unit, weight, images, attributes, channels,
       channel_listing, is_active, created_by, created_at, updated_at
FROM products
WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE
FOR UPDATE;

-- name: DeductProductStock :one
UPDATE products
SET stock_quantity = stock_quantity - sqlc.arg(qty),
    updated_at     = NOW()
WHERE id = sqlc.arg(id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING stock_quantity;

-- name: ListProducts :many
SELECT id, tenant_id, category_id, sku, barcode, name, description,
       price, cost_price, compare_at_price, stock_quantity, min_stock_level,
       track_inventory, unit, weight, images, attributes, channels,
       channel_listing, is_active, created_by, created_at, updated_at
FROM products
WHERE tenant_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProducts :one
SELECT COUNT(*) FROM products
WHERE tenant_id = $1 AND is_active = TRUE;

-- name: SearchProducts :many
SELECT id, tenant_id, category_id, sku, barcode, name, description,
       price, cost_price, compare_at_price, stock_quantity, min_stock_level,
       track_inventory, unit, weight, images, attributes, channels,
       channel_listing, is_active, created_by, created_at, updated_at
FROM products
WHERE tenant_id = $1
  AND is_active = TRUE
  AND (name ILIKE '%' || sqlc.arg(query) || '%'
       OR sku ILIKE '%' || sqlc.arg(query) || '%'
       OR barcode = sqlc.arg(query))
ORDER BY name
LIMIT $2;
