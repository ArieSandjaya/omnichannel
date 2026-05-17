-- name: GetRecentOrders :many
SELECT * FROM orders
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1
LIMIT 1;

-- name: GetOrderByNumber :one
SELECT * FROM orders
WHERE tenant_id = $1 AND order_number = $2
LIMIT 1;

-- name: GetOrdersByStatus :many
SELECT * FROM orders
WHERE tenant_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: GetTodayRevenue :one
SELECT COALESCE(SUM(total_amount), 0)::BIGINT AS total
FROM orders
WHERE tenant_id = $1
  AND created_at >= CURRENT_DATE
  AND payment_status = 'paid';

-- name: GetTodayOrderCount :one
SELECT COUNT(*)::INT AS count
FROM orders
WHERE tenant_id = $1
  AND created_at >= CURRENT_DATE;

-- name: CreateOrder :one
INSERT INTO orders (
  tenant_id, order_number, channel, status,
  line_items, subtotal, discount_amount, total_amount,
  payment_method, payment_status, created_by
) VALUES (
  $1, $2, $3, $4,
  $5, $6, $7, $8,
  $9, $10, $11
)
RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
