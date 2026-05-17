package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// BiteshipWebhookPayload is the shape of a Biteship status webhook body.
type BiteshipWebhookPayload struct {
	ID        string `json:"id"` // Biteship order ID
	Status    string `json:"status"`
	WaybillID string `json:"waybill_id"`
	Courier   struct {
		Company string `json:"company"`
	} `json:"courier"`
}

// ShippingStatusUpdater is the interface fulfilled by service.ShippingService.
type ShippingStatusUpdater interface {
	UpdateShippingStatus(ctx context.Context, biteshipOrderID, status, waybillID string) error
}

// BiteshipWebhookHandler processes TypeWebhookBiteship Asynq tasks.
type BiteshipWebhookHandler struct {
	shippingSvc ShippingStatusUpdater
}

func NewBiteshipWebhookHandler(svc ShippingStatusUpdater) *BiteshipWebhookHandler {
	return &BiteshipWebhookHandler{shippingSvc: svc}
}

func (h *BiteshipWebhookHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload BiteshipWebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal Biteship webhook payload: %w", err)
	}

	slog.Info("processing Biteship webhook",
		"biteship_order_id", payload.ID,
		"status", payload.Status,
		"waybill_id", payload.WaybillID,
	)

	return h.shippingSvc.UpdateShippingStatus(ctx, payload.ID, payload.Status, payload.WaybillID)
}
