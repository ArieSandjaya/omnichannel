-- name: CreateInventoryLog :one
INSERT INTO inventory_log (
    id, tenant_id, product_id, type, quantity_change,
    quantity_before, quantity_after, reference_type, reference_id,
    channel, notes, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12
)
RETURNING *;

-- name: ListInventoryLogByProduct :many
SELECT id, tenant_id, product_id, type, quantity_change,
       quantity_before, quantity_after, reference_type, reference_id,
       channel, notes, created_by, created_at
FROM inventory_log
WHERE tenant_id = $1 AND product_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountInventoryLogByProduct :one
SELECT COUNT(*) FROM inventory_log
WHERE tenant_id = $1 AND product_id = $2;

-- name: ListInventoryLogByTenant :many
SELECT id, tenant_id, product_id, type, quantity_change,
       quantity_before, quantity_after, reference_type, reference_id,
       channel, notes, created_by, created_at
FROM inventory_log
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListInventoryLogByReference :many
SELECT id, tenant_id, product_id, type, quantity_change,
       quantity_before, quantity_after, reference_type, reference_id,
       channel, notes, created_by, created_at
FROM inventory_log
WHERE tenant_id = $1
  AND reference_type = $2
  AND reference_id   = $3
ORDER BY created_at;
