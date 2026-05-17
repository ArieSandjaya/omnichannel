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

type POSHandler struct {
	svc      *service.POSService
	resp     *response.Responder
	validate *validator.Validate
}

func NewPOSHandler(svc *service.POSService, resp *response.Responder) *POSHandler {
	return &POSHandler{svc: svc, resp: resp, validate: validator.New()}
}

// CreateTransaction handles POST /api/v1/pos/transactions
func (h *POSHandler) CreateTransaction(c *gin.Context) {
	var req dto.POSTransactionRequest
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

	result, err := h.svc.ProcessTransaction(c.Request.Context(), req, tenantID, userID)
	if err != nil {
		h.resp.Error(c, http.StatusConflict, err.Error(), nil)
		return
	}
	h.resp.Success(c, http.StatusCreated, "transaksi POS berhasil", result)
}
