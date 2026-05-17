package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ariesandjaya/omnichannel/internal/dto"
	"github.com/ariesandjaya/omnichannel/internal/middleware"
	"github.com/ariesandjaya/omnichannel/internal/service"
)

type ShippingHandler struct {
	svc *service.ShippingService
}

func NewShippingHandler(svc *service.ShippingService) *ShippingHandler {
	return &ShippingHandler{svc: svc}
}

// GetRates handles POST /api/v1/shipping/rates
func (h *ShippingHandler) GetRates(c *gin.Context) {
	var req dto.GetRatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.GetRates(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// CreateShipment handles POST /api/v1/orders/:orderID/shipping
func (h *ShippingHandler) CreateShipment(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	orderID, err := uuid.Parse(c.Param("orderID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	var req dto.CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreateShipment(c.Request.Context(), tenantID, orderID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// Track handles GET /api/v1/shipping/track/:waybillID
func (h *ShippingHandler) Track(c *gin.Context) {
	waybillID := c.Param("waybillID")
	courierCode := c.Query("courier")

	resp, err := h.svc.GetTrackingInfo(c.Request.Context(), waybillID, courierCode)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
