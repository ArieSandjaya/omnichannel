package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/ariesandjaya/omnichannel/internal/broker"
	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/middleware"
	"github.com/ariesandjaya/omnichannel/internal/service"
)

// RouterDeps bundles all dependencies needed to configure the Gin router.
type RouterDeps struct {
	Config      *config.Config
	PaymentSvc  *service.PaymentService
	ShippingSvc *service.ShippingService
	SSEBroker   *broker.SSEBroker
	AsynqClient *asynq.Client
}

// SetupRouter configures all Gin routes and returns the engine.
func SetupRouter(deps RouterDeps) *gin.Engine {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// ── Health check ──────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── Webhook routes ────────────────────────────────────────────────────
	// No auth middleware — each endpoint has its own token/HMAC verification.
	webhookHandler := NewWebhookHandler(
		deps.AsynqClient,
		deps.Config.Xendit.WebhookToken,
		deps.Config.Biteship.WebhookSecret,
	)
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/xendit/qris", webhookHandler.HandleXenditQRIS)
		webhooks.POST("/xendit/virtual-account", webhookHandler.HandleXenditVA)
		webhooks.POST("/biteship", webhookHandler.HandleBiteship)
	}

	// ── Authenticated API routes ──────────────────────────────────────────
	authMW := middleware.Auth(deps.Config.JWTSecret)
	api := r.Group("/api/v1", authMW)
	{
		paymentH := NewPaymentHandler(deps.PaymentSvc)
		shippingH := NewShippingHandler(deps.ShippingSvc)

		orders := api.Group("/orders/:orderID")
		{
			orders.POST("/payment/qris", paymentH.CreateQRIS)
			orders.POST("/payment/virtual-account", paymentH.CreateVA)
			orders.POST("/shipping", shippingH.CreateShipment)
		}

		api.POST("/shipping/rates", shippingH.GetRates)
		api.GET("/shipping/track/:waybillID", shippingH.Track)
	}

	// ── SSE stream ────────────────────────────────────────────────────────
	sseHandler := NewSSEHandler(deps.SSEBroker)
	sse := r.Group("/sse", middleware.Auth(deps.Config.JWTSecret))
	{
		sse.GET("/stock", sseHandler.Stock)
	}

	return r
}
