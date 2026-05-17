-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1
LIMIT 1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants
WHERE slug = $1
LIMIT 1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY created_at DESC;

-- name: CreateTenant :one
INSERT INTO tenants (name, slug, business_type, subscription_plan, settings, logo_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTenantSettings :one
UPDATE tenants
SET settings = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateTenantLogo :one
UPDATE tenants
SET logo_url = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
