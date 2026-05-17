package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// XenditQRISPayload is the shape of the Xendit QRIS webhook body.
type XenditQRISPayload struct {
	ID         string  `json:"id"`
	ExternalID string  `json:"external_id"`
	Status     string  `json:"status"` // "ACTIVE" = paid
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	QRString   string  `json:"qr_string"`
}

// XenditVAPayload is the shape of the Xendit Fixed VA payment webhook body.
type XenditVAPayload struct {
	ID            string  `json:"id"`
	ExternalID    string  `json:"external_id"`
	AccountNumber string  `json:"account_number"`
	BankCode      string  `json:"bank_code"`
	PaymentAmount float64 `json:"payment_amount"`
	Status        string  `json:"status"` // "PAID"
	PaymentID     string  `json:"payment_id"`
}

// PaymentSuccessHandler is the interface fulfilled by service.PaymentService.
type PaymentSuccessHandler interface {
	HandlePaymentSuccess(ctx context.Context, externalID string, rawPayload json.RawMessage) error
}

// XenditQRISWebhookHandler processes TypeWebhookXenditQRIS Asynq tasks.
type XenditQRISWebhookHandler struct {
	paymentSvc PaymentSuccessHandler
}

func NewXenditQRISWebhookHandler(svc PaymentSuccessHandler) *XenditQRISWebhookHandler {
	return &XenditQRISWebhookHandler{paymentSvc: svc}
}

func (h *XenditQRISWebhookHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload XenditQRISPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal QRIS webhook payload: %w", err)
	}

	slog.Info("processing QRIS webhook",
		"external_id", payload.ExternalID,
		"status", payload.Status,
	)

	// Xendit QRIS "paid" status is "ACTIVE"
	if payload.Status != "ACTIVE" {
		slog.Info("QRIS non-paid status, skipping", "status", payload.Status)
		return nil
	}

	return h.paymentSvc.HandlePaymentSuccess(ctx, payload.ExternalID, t.Payload())
}

// XenditVAWebhookHandler processes TypeWebhookXenditVA Asynq tasks.
type XenditVAWebhookHandler struct {
	paymentSvc PaymentSuccessHandler
}

func NewXenditVAWebhookHandler(svc PaymentSuccessHandler) *XenditVAWebhookHandler {
	return &XenditVAWebhookHandler{paymentSvc: svc}
}

func (h *XenditVAWebhookHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload XenditVAPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal VA webhook payload: %w", err)
	}

	slog.Info("processing VA webhook",
		"external_id", payload.ExternalID,
		"bank_code", payload.BankCode,
		"status", payload.Status,
	)

	if payload.Status != "PAID" {
		slog.Info("VA non-paid status, skipping", "status", payload.Status)
		return nil
	}

	return h.paymentSvc.HandlePaymentSuccess(ctx, payload.ExternalID, t.Payload())
}
