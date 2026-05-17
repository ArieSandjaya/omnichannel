-- name: GetProductsByTenant :many
SELECT * FROM products
WHERE tenant_id = $1 AND is_active = TRUE
ORDER BY name ASC;

-- name: SearchProducts :many
SELECT * FROM products
WHERE tenant_id = $1
  AND is_active = TRUE
  AND (name ILIKE '%' || $2 || '%' OR sku ILIKE '%' || $2 || '%')
ORDER BY name ASC
LIMIT 50;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1
LIMIT 1;

-- name: GetProductBySKU :one
SELECT * FROM products
WHERE tenant_id = $1 AND sku = $2
LIMIT 1;

-- name: GetLowStockProducts :many
SELECT * FROM products
WHERE tenant_id = $1
  AND is_active = TRUE
  AND track_inventory = TRUE
  AND stock_quantity <= min_stock_level
ORDER BY stock_quantity ASC;

-- name: GetProductsForStorefront :many
SELECT * FROM products
WHERE tenant_id = $1
  AND is_active = TRUE
  AND 'website' = ANY(SELECT jsonb_array_elements_text(channels))
ORDER BY name ASC;

-- name: UpdateProductStock :one
UPDATE products
SET stock_quantity = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateProduct :one
INSERT INTO products (
  tenant_id, category_id, sku, barcode, name, description,
  price, cost_price, stock_quantity, min_stock_level,
  unit, image_url, channels, is_active, created_by
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10,
  $11, $12, $13, $14, $15
)
RETURNING *;
