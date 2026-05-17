package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ariesandjaya/omnichannel/internal/broker"
	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/ariesandjaya/omnichannel/internal/middleware"
	"github.com/ariesandjaya/omnichannel/templates/pages"
	storefrontpage "github.com/ariesandjaya/omnichannel/templates/pages/storefront"
	pospartial "github.com/ariesandjaya/omnichannel/templates/partials"
)

func SetupRouter(cfg *config.Config, sseBroker *broker.SSEBroker) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	r.Static("/static", "./static")

	// ── Middleware providers (inject mock data in dev mode) ─────────
	tenantProvider := middleware.TenantProvider(MockTenantBySlug)
	tenantUserProvider := middleware.TenantUserProvider(func(tenantID string) (*domain.Tenant, *domain.User) {
		// In production: fetch from DB. In mock mode: return demo data.
		return MockTenants[0], MockUsers[0]
	})

	// ── Public ──────────────────────────────────────────────────────
	r.GET("/login", func(c *gin.Context) {
		renderPage(c, 200, pages.Login(""))
	})

	// Storefront — tenant identified from subdomain via middleware
	sf := r.Group("/")
	sf.Use(middleware.StorefrontMiddleware(tenantProvider))
	{
		sf.GET("/", func(c *gin.Context) {
			slug := c.GetString("storefront_slug")
			tenant := MockTenantBySlug(slug)
			renderPage(c, 200, storefrontpage.StorefrontHome(storefrontpage.StorefrontHomeProps{
				Tenant:   tenant,
				Products: activeProducts(),
			}))
		})
	}

	// ── Protected admin routes ───────────────────────────────────────
	admin := r.Group("/")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret, cfg.MockMode))
	admin.Use(middleware.AdminTenantMiddleware(tenantUserProvider))
	{
		admin.GET("/dashboard", func(c *gin.Context) {
			tenant, user := getTenantAndUser(c)
			stats := MockDashboardStats()
			renderPage(c, 200, pages.Dashboard(pages.DashboardProps{
				Tenant:       tenant,
				User:         user,
				Stats:        pages.DashboardStats(stats),
				RecentOrders: MockOrders,
				LowStock:     LowStockProducts(),
				Channels:     toPageChannels(mockChannelStatuses()),
			}))
		})

		admin.GET("/pos", func(c *gin.Context) {
			tenant, user := getTenantAndUser(c)
			renderPage(c, 200, pages.POS(pages.POSProps{
				Tenant:   tenant,
				User:     user,
				Products: MockProducts,
			}))
		})

		// HTMX partials for POS
		posAPI := admin.Group("/api/pos")
		{
			posAPI.GET("/products/search", func(c *gin.Context) {
				q := c.Query("q")
				renderPartial(c, pospartial.ProductSearchResults(searchProducts(q)))
			})

			posAPI.POST("/cart/add", func(c *gin.Context) {
				// Session-based cart in production; return empty for mock
				renderPartial(c, pospartial.CartItems(pospartial.CartSummary{}))
			})

			posAPI.DELETE("/cart/item/:id", func(c *gin.Context) {
				renderPartial(c, pospartial.CartItems(pospartial.CartSummary{}))
			})

			posAPI.DELETE("/cart/clear", func(c *gin.Context) {
				renderPartial(c, pospartial.CartItems(pospartial.CartSummary{}))
			})
		}

		// SSE real-time stock stream
		sseH := NewSSEHandler(sseBroker)
		admin.GET("/sse/stock", sseH.StockStream)
	}

	return r
}

// activeProducts returns only active products (for storefront).
func activeProducts() []domain.Product {
	var result []domain.Product
	for _, p := range MockProducts {
		if p.IsActive {
			result = append(result, p)
		}
	}
	return result
}

// searchProducts filters MockProducts by name or SKU (case-insensitive substring).
func searchProducts(q string) []domain.Product {
	if q == "" {
		return MockProducts
	}
	q = strings.ToLower(q)
	var result []domain.Product
	for _, p := range MockProducts {
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.SKU), q) {
			result = append(result, p)
		}
	}
	return result
}

// toPageChannels converts handler.ChannelStatus to pages.ChannelStatus.
func toPageChannels(channels []ChannelStatus) []pages.ChannelStatus {
	out := make([]pages.ChannelStatus, len(channels))
	for i, ch := range channels {
		out[i] = pages.ChannelStatus{
			Name:      ch.Name,
			Connected: ch.Connected,
			LastSync:  ch.LastSync,
		}
	}
	return out
}
