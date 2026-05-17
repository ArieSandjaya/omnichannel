package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// StockSyncPayload is the task payload for stock synchronisation to marketplaces.
type StockSyncPayload struct {
	TenantID  string `json:"tenant_id"`
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	// NewQuantity == -1 means the worker should re-read the current quantity from DB.
	NewQuantity int `json:"new_quantity"`
}

// StockSyncHandler processes TypeStockSync Asynq tasks.
type StockSyncHandler struct{}

func NewStockSyncHandler() *StockSyncHandler {
	return &StockSyncHandler{}
}

func (h *StockSyncHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p StockSyncPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal StockSyncPayload: %w", err)
	}

	slog.Info("stock sync task received",
		"tenant_id", p.TenantID,
		"product_id", p.ProductID,
		"sku", p.SKU,
		"new_quantity", p.NewQuantity,
	)

	// TODO: implement per-marketplace sync
	// - Shopee: PUT /api/v2/product/update_stock
	// - Tokopedia: POST /inventory/v1/fs/{fs_id}/product/inventory/update
	// - TikTok Shop: POST /api/products/stocks
	// Each marketplace call should be idempotent and log failures without
	// returning an error so the Asynq task is not retried unnecessarily.

	return nil
}
