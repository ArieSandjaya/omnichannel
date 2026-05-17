package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ariesandjaya/omnichannel/internal/domain"
)

// TenantProvider looks up a tenant by subdomain slug.
type TenantProvider func(slug string) *domain.Tenant

// TenantUserProvider loads tenant + user for an authenticated request.
type TenantUserProvider func(tenantID string) (*domain.Tenant, *domain.User)

// StorefrontMiddleware extracts the tenant slug from the Host header subdomain
// and populates gin.Context["tenant"] and gin.Context["storefront_slug"].
// e.g. "batik-nusantara.myapp.com" → slug = "batik-nusantara"
func StorefrontMiddleware(tenantFn TenantProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := extractSlugFromHost(c.Request.Host)
		tenant := tenantFn(slug)
		c.Set("tenant", tenant)
		c.Set("storefront_slug", slug)
		c.Next()
	}
}

// AdminTenantMiddleware populates gin.Context with the authenticated tenant and user.
// In production, providerFn fetches from the database using the JWT tenant_id claim.
func AdminTenantMiddleware(providerFn TenantUserProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		tenant, user := providerFn(tenantID)
		if tenant != nil {
			c.Set("tenant", tenant)
		}
		if user != nil {
			c.Set("user", user)
		}
		c.Next()
	}
}

// GetTenant retrieves the tenant from gin.Context.
func GetTenant(c *gin.Context) *domain.Tenant {
	v, exists := c.Get("tenant")
	if !exists {
		return nil
	}
	t, _ := v.(*domain.Tenant)
	return t
}

// GetUser retrieves the current user from gin.Context.
func GetUser(c *gin.Context) *domain.User {
	v, exists := c.Get("user")
	if !exists {
		return nil
	}
	u, _ := v.(*domain.User)
	return u
}

func extractSlugFromHost(host string) string {
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	parts := strings.Split(host, ".")
	if len(parts) >= 2 && parts[0] != "www" && parts[0] != "localhost" {
		return parts[0]
	}
	return "demo"
}
