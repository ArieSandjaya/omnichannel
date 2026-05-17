// Package repository provides the database access layer.
// Before running `make sqlc`, this file contains a raw pgx implementation
// that matches the same interface. After sqlc generates db/sqlc/, the
// SqlcAdapter (in sqlc_adapter.go) wraps the generated code instead.
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ariesandjaya/omnichannel/internal/service"
)

// RawAdapter implements service.DBQuerier directly against pgxpool using raw SQL.
// This is the bootstrap adapter used before `make sqlc` generates the type-safe layer.
type RawAdapter struct {
	pool *pgxpool.Pool
}

func NewRawAdapter(pool *pgxpool.Pool) *RawAdapter {
	return &RawAdapter{pool: pool}
}

// ── Payment queries ───────────────────────────────────────────────────────────

func (a *RawAdapter) GetOrderByIDRaw(ctx context.Context, id, tenantID uuid.UUID) (service.OrderRow, error) {
	const q = `
		SELECT id, tenant_id, order_number, status, payment_status,
		       customer_name, subtotal, shipping_cost, total, line_items,
		       payment_info, shipping_info
		FROM orders
		WHERE id = $1 AND tenant_id = $2
	`
	return a.scanOrder(a.pool.QueryRow(ctx, q, id, tenantID))
}

func (a *RawAdapter) GetOrderByExternalPaymentIDRaw(ctx context.Context, externalID string) (service.OrderRow, error) {
	const q = `
		SELECT id, tenant_id, order_number, status, payment_status,
		       customer_name, subtotal, shipping_cost, total, line_items,
		       payment_info, shipping_info
		FROM orders
		WHERE payment_info->>'external_id' = $1
		LIMIT 1
	`
	return a.scanOrder(a.pool.QueryRow(ctx, q, externalID))
}

func (a *RawAdapter) GetOrderByBiteshipOrderIDRaw(ctx context.Context, biteshipOrderID string) (service.OrderRow, error) {
	const q = `
		SELECT id, tenant_id, order_number, status, payment_status,
		       customer_name, subtotal, shipping_cost, total, line_items,
		       payment_info, shipping_info
		FROM orders
		WHERE shipping_info->>'biteship_order_id' = $1
		LIMIT 1
	`
	return a.scanOrder(a.pool.QueryRow(ctx, q, biteshipOrderID))
}

func (a *RawAdapter) UpdatePaymentInfoRaw(ctx context.Context, id, tenantID uuid.UUID, status string, info json.RawMessage) error {
	const q = `
		UPDATE orders
		SET payment_status = $3, payment_info = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := a.pool.Exec(ctx, q, id, tenantID, status, info)
	return err
}

func (a *RawAdapter) ConfirmPaymentAndProcessOrderRaw(ctx context.Context, id, tenantID uuid.UUID, paymentStatus, status string, info json.RawMessage) error {
	const q = `
		UPDATE orders
		SET payment_status = $3, status = $4, payment_info = $5, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := a.pool.Exec(ctx, q, id, tenantID, paymentStatus, status, info)
	return err
}

func (a *RawAdapter) DeductStockRaw(ctx context.Context, tenantID, productID, orderID uuid.UUID, qty int32, reason string) error {
	const q = `SELECT deduct_stock($1, $2, $3, $4, $5)`
	_, err := a.pool.Exec(ctx, q, tenantID, productID, orderID, qty, reason)
	return err
}

// ── Shipping queries ──────────────────────────────────────────────────────────

func (a *RawAdapter) UpdateShippingInfoRaw(ctx context.Context, id, tenantID uuid.UUID, info json.RawMessage) error {
	const q = `
		UPDATE orders
		SET shipping_info = $3, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := a.pool.Exec(ctx, q, id, tenantID, info)
	return err
}

// ── scanner ───────────────────────────────────────────────────────────────────

func (a *RawAdapter) scanOrder(row pgx.Row) (service.OrderRow, error) {
	var o service.OrderRow
	var paymentInfo, shippingInfo []byte
	err := row.Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &o.Status, &o.PaymentStatus,
		&o.CustomerName, &o.Subtotal, &o.ShippingCost, &o.Total,
		&o.LineItems, &paymentInfo, &shippingInfo,
	)
	if err != nil {
		return service.OrderRow{}, fmt.Errorf("scan order: %w", err)
	}
	if paymentInfo != nil {
		o.PaymentInfo = json.RawMessage(paymentInfo)
	}
	if shippingInfo != nil {
		o.ShippingInfo = json.RawMessage(shippingInfo)
	}
	return o, nil
}
