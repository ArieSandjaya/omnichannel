-- name: CreateProduct :one
INSERT INTO products (
    tenant_id, category_id, sku, name, description, price, cost_price,
    stock_quantity, min_stock_level, unit, weight_grams, image_url
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1 AND tenant_id = $2;

-- name: GetProductBySKU :one
SELECT * FROM products
WHERE sku = $1 AND tenant_id = $2;

-- name: ListProducts :many
SELECT * FROM products
WHERE tenant_id = $1 AND is_active = TRUE
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: ListLowStockProducts :many
SELECT * FROM products
WHERE tenant_id = $1
  AND track_inventory = TRUE
  AND stock_quantity <= $2
  AND is_active = TRUE
ORDER BY stock_quantity ASC;

-- name: DeductStock :exec
-- Calls the atomic PostgreSQL function that uses FOR UPDATE lock
SELECT deduct_stock($1, $2, $3, $4, $5);

-- name: UpdateProductStock :one
UPDATE products
SET stock_quantity = $3,
    updated_at     = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;
