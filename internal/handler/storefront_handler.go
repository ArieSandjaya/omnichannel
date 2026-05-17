package handler

// StorefrontHandler exists as a placeholder for production DB-backed storefront logic.
// In mock mode, the storefront routes in router.go use MockTenantBySlug directly.
type StorefrontHandler struct{}

func NewStorefrontHandler() *StorefrontHandler {
	return &StorefrontHandler{}
}
