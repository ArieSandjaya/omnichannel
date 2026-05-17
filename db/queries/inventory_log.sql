-- name: ListInventoryLogByProduct :many
SELECT * FROM inventory_log
WHERE tenant_id = $1 AND product_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListInventoryLogByOrder :many
SELECT * FROM inventory_log
WHERE tenant_id = $1 AND order_id = $2
ORDER BY created_at ASC;
