package handler

import (
	"net/http"

	"github.com/ariesandjaya/omnichannel/internal/dto"
	"github.com/ariesandjaya/omnichannel/internal/service"
	"github.com/ariesandjaya/omnichannel/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CheckoutHandler struct {
	svc      *service.CheckoutService
	resp     *response.Responder
	validate *validator.Validate
}

func NewCheckoutHandler(svc *service.CheckoutService, resp *response.Responder) *CheckoutHandler {
	return &CheckoutHandler{svc: svc, resp: resp, validate: validator.New()}
}

// Process handles POST /api/v1/checkout
func (h *CheckoutHandler) Process(c *gin.Context) {
	var req dto.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.Error(c, http.StatusBadRequest, "request tidak valid", err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.resp.Error(c, http.StatusUnprocessableEntity, "validasi gagal", err)
		return
	}

	tenantID := uuid.MustParse(c.GetString("tenant_id"))
	userID := uuid.MustParse(c.GetString("user_id"))

	result, err := h.svc.Process(c.Request.Context(), req, tenantID, userID)
	if err != nil {
		h.resp.Error(c, http.StatusConflict, err.Error(), nil)
		return
	}
	h.resp.Success(c, http.StatusCreated, "checkout berhasil", result)
}
