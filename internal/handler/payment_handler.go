package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ariesandjaya/omnichannel/internal/dto"
	"github.com/ariesandjaya/omnichannel/internal/middleware"
	"github.com/ariesandjaya/omnichannel/internal/service"
)

type PaymentHandler struct {
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// CreateQRIS handles POST /api/v1/orders/:orderID/payment/qris
func (h *PaymentHandler) CreateQRIS(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	orderID, err := uuid.Parse(c.Param("orderID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	var req dto.CreateQRISRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreateQRISPayment(c.Request.Context(), tenantID, orderID,
		req.CustomerName, req.CustomerEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// CreateVA handles POST /api/v1/orders/:orderID/payment/virtual-account
func (h *PaymentHandler) CreateVA(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	orderID, err := uuid.Parse(c.Param("orderID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	var req dto.CreateVARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreateVAPayment(c.Request.Context(), tenantID, orderID,
		req.BankCode, req.CustomerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}
