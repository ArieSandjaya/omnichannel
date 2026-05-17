-- name: CreateOrder :one
INSERT INTO orders (
    id, tenant_id, order_number, channel, channel_order_id,
    status, customer_id, customer_info, line_items,
    subtotal, discount_amount, tax_amount, shipping_amount, total_amount,
    payment_method, payment_status, payment_info,
    shipping_address, shipping_info, notes, metadata,
    created_by, completed_at, cancelled_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16, $17,
    $18, $19, $20, $21,
    $22, $23, $24
)
RETURNING *;

-- name: GetOrderByID :one
SELECT id, tenant_id, order_number, channel, channel_order_id,
       status, customer_id, customer_info, line_items,
       subtotal, discount_amount, tax_amount, shipping_amount, total_amount,
       payment_method, payment_status, payment_info,
       shipping_address, shipping_info, notes, metadata,
       created_by, completed_at, cancelled_at, created_at, updated_at
FROM orders
WHERE id = $1 AND tenant_id = $2;

-- name: ListOrdersByTenant :many
SELECT id, tenant_id, order_number, channel, channel_order_id,
       status, customer_id, customer_info, line_items,
       subtotal, discount_amount, tax_amount, shipping_amount, total_amount,
       payment_method, payment_status, payment_info,
       shipping_address, shipping_info, notes, metadata,
       created_by, completed_at, cancelled_at, created_at, updated_at
FROM orders
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListOrdersByChannel :many
SELECT id, tenant_id, order_number, channel, channel_order_id,
       status, customer_id, customer_info, line_items,
       subtotal, discount_amount, tax_amount, shipping_amount, total_amount,
       payment_method, payment_status, payment_info,
       shipping_address, shipping_info, notes, metadata,
       created_by, completed_at, cancelled_at, created_at, updated_at
FROM orders
WHERE tenant_id = $1 AND channel = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status       = sqlc.arg(status),
    updated_at   = NOW(),
    completed_at = CASE WHEN sqlc.arg(status) = 'completed' THEN NOW() ELSE completed_at END,
    cancelled_at = CASE WHEN sqlc.arg(status) = 'cancelled' THEN NOW() ELSE cancelled_at END
WHERE id = sqlc.arg(id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: UpdateOrderPaymentStatus :one
UPDATE orders
SET payment_status = sqlc.arg(payment_status),
    payment_info   = sqlc.arg(payment_info),
    updated_at     = NOW()
WHERE id = sqlc.arg(id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;
