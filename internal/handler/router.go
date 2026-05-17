package handler

import (
	"net/http"

	"github.com/ariesandjaya/omnichannel/internal/middleware"
	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	Auth     *middleware.AuthMiddleware
	Checkout *CheckoutHandler
	POS      *POSHandler
}

func NewRouter(deps RouterDeps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(deps.Auth.Logger())
	r.Use(deps.Auth.CORS())

	// Health check (no auth)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	// Protected routes — requires valid JWT + resolved tenant
	protected := v1.Group("")
	protected.Use(deps.Auth.JWT(), deps.Auth.ResolveTenant())
	{
		// Web / e-commerce checkout
		protected.POST("/checkout", deps.Checkout.Process)

		// POS terminal — only cashier, manager, or owner may create transactions
		pos := protected.Group("/pos")
		pos.Use(deps.Auth.RequireRole("cashier", "manager", "owner", "super_admin"))
		{
			pos.POST("/transactions", deps.POS.CreateTransaction)
		}
	}

	return r
}
