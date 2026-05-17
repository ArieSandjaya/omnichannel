-- name: CreateOrder :one
INSERT INTO orders (
    tenant_id, order_number, channel, customer_name, customer_email,
    customer_phone, subtotal, shipping_cost, total, notes, line_items
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1 AND tenant_id = $2;

-- name: GetOrderByOrderNumber :one
SELECT * FROM orders
WHERE order_number = $1 AND tenant_id = $2;

-- name: GetOrderByExternalPaymentID :one
-- Used by webhook handler (no tenant_id filter; runs with BYPASSRLS role)
SELECT * FROM orders
WHERE payment_info->>'external_id' = $1
LIMIT 1;

-- name: GetOrderByBiteshipOrderID :one
-- Used by Biteship webhook handler (no tenant_id filter; runs with BYPASSRLS role)
SELECT * FROM orders
WHERE shipping_info->>'biteship_order_id' = $1
LIMIT 1;

-- name: UpdatePaymentInfo :one
UPDATE orders
SET payment_status = $3,
    payment_info   = $4,
    updated_at     = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status     = $3,
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateShippingInfo :one
UPDATE orders
SET shipping_info = $3,
    updated_at    = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: ConfirmPaymentAndProcessOrder :one
-- Atomic: set both payment_status=paid and status=processing in one UPDATE
UPDATE orders
SET payment_status = $3,
    status         = $4,
    payment_info   = $5,
    updated_at     = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: ListOrders :many
SELECT * FROM orders
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListOrdersByStatus :many
SELECT * FROM orders
WHERE tenant_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
