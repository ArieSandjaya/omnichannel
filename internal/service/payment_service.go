package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/ariesandjaya/omnichannel/internal/broker"
	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/ariesandjaya/omnichannel/internal/dto"
	"github.com/ariesandjaya/omnichannel/internal/gateway"
	"github.com/ariesandjaya/omnichannel/internal/worker"
)

// OrderRow is the minimal order shape returned by the repository layer.
type OrderRow struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	OrderNumber   string
	Status        string
	PaymentStatus string
	CustomerName  string
	Subtotal      int64
	ShippingCost  int64
	Total         int64
	LineItems     json.RawMessage
	PaymentInfo   json.RawMessage
	ShippingInfo  json.RawMessage
}

// PaymentQuerier is the DB interface required by PaymentService.
type PaymentQuerier interface {
	GetOrderByIDRaw(ctx context.Context, id, tenantID uuid.UUID) (OrderRow, error)
	GetOrderByExternalPaymentIDRaw(ctx context.Context, externalID string) (OrderRow, error)
	UpdatePaymentInfoRaw(ctx context.Context, id, tenantID uuid.UUID, status string, info json.RawMessage) error
	ConfirmPaymentAndProcessOrderRaw(ctx context.Context, id, tenantID uuid.UUID, paymentStatus, status string, info json.RawMessage) error
	DeductStockRaw(ctx context.Context, tenantID, productID, orderID uuid.UUID, qty int32, reason string) error
}

// PaymentService handles Xendit QRIS and Virtual Account payment creation,
// webhook processing, and atomic stock deduction.
type PaymentService struct {
	queries  PaymentQuerier
	xendit   gateway.XenditGateway
	broker   *broker.SSEBroker
	asynq    *asynq.Client
	cfg      *config.Config
}

func NewPaymentService(
	queries PaymentQuerier,
	xenditGw gateway.XenditGateway,
	sseBroker *broker.SSEBroker,
	asynqClient *asynq.Client,
	cfg *config.Config,
) *PaymentService {
	return &PaymentService{
		queries: queries,
		xendit:  xenditGw,
		broker:  sseBroker,
		asynq:   asynqClient,
		cfg:     cfg,
	}
}

// generateExternalID embeds tenantID so webhooks can resolve the order without a JWT.
// Format: omni-{tenantUUID}-{orderUUID}-{unixmilli}
func generateExternalID(tenantID, orderID uuid.UUID) string {
	return fmt.Sprintf("omni-%s-%s-%d", tenantID, orderID, time.Now().UnixMilli())
}

// CreateQRISPayment creates a Xendit QRIS Dynamic QR Code for the order total.
// Idempotent: returns the existing payment info if one already exists for this order.
func (s *PaymentService) CreateQRISPayment(
	ctx context.Context,
	tenantID, orderID uuid.UUID,
	customerName, customerEmail string,
) (*dto.PaymentResponse, error) {
	order, err := s.queries.GetOrderByIDRaw(ctx, orderID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Idempotency
	if order.PaymentInfo != nil {
		var existing domain.PaymentInfo
		if json.Unmarshal(order.PaymentInfo, &existing) == nil && existing.ExternalID != "" {
			return paymentInfoToResponse(&existing), nil
		}
	}

	externalID := generateExternalID(tenantID, orderID)
	expiresAt := time.Now().Add(time.Duration(s.cfg.Xendit.QRISExpirySeconds) * time.Second)

	resp, err := s.xendit.CreateQRIS(ctx, externalID, s.cfg.Xendit.QRISCallbackURL, float64(order.Total))
	if err != nil {
		return nil, err
	}

	payInfo := domain.PaymentInfo{
		ExternalID:  externalID,
		PaymentType: domain.PaymentTypeQRIS,
		XenditID:    resp.XenditID,
		Amount:      order.Total,
		Status:      "pending",
		QRISString:  resp.QRString,
		ExpiresAt:   &expiresAt,
		TenantID:    tenantID,
	}
	payInfoJSON, _ := json.Marshal(payInfo)

	if err := s.queries.UpdatePaymentInfoRaw(ctx, orderID, tenantID, "pending", payInfoJSON); err != nil {
		return nil, fmt.Errorf("persist QRIS payment: %w", err)
	}

	slog.Info("QRIS payment created",
		"external_id", externalID, "order_id", orderID, "amount", order.Total)
	return paymentInfoToResponse(&payInfo), nil
}

// CreateVAPayment creates a Xendit Fixed Virtual Account for the order total.
// Idempotent: returns the existing payment info if one already exists for this order.
func (s *PaymentService) CreateVAPayment(
	ctx context.Context,
	tenantID, orderID uuid.UUID,
	bankCode, customerName string,
) (*dto.PaymentResponse, error) {
	bankCode = strings.ToUpper(strings.TrimSpace(bankCode))
	if !isValidVABank(bankCode, s.cfg.Xendit.SupportedVABanks) {
		return nil, fmt.Errorf("unsupported bank %q; supported: %s",
			bankCode, strings.Join(s.cfg.Xendit.SupportedVABanks, ", "))
	}

	order, err := s.queries.GetOrderByIDRaw(ctx, orderID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if order.PaymentInfo != nil {
		var existing domain.PaymentInfo
		if json.Unmarshal(order.PaymentInfo, &existing) == nil && existing.ExternalID != "" {
			return paymentInfoToResponse(&existing), nil
		}
	}

	externalID := generateExternalID(tenantID, orderID)
	expiresAt := time.Now().Add(time.Duration(s.cfg.Xendit.VAExpiryHours) * time.Hour)
	vaName := sanitizeVAName(customerName)

	resp, err := s.xendit.CreateFixedVA(ctx, externalID, bankCode, vaName, float64(order.Total), expiresAt)
	if err != nil {
		return nil, err
	}

	payInfo := domain.PaymentInfo{
		ExternalID:  externalID,
		PaymentType: domain.PaymentTypeVirtualAccount,
		XenditID:    resp.XenditID,
		Amount:      order.Total,
		Status:      "pending",
		VANumber:    resp.AccountNumber,
		BankCode:    bankCode,
		VAName:      vaName,
		ExpiresAt:   &expiresAt,
		TenantID:    tenantID,
	}
	payInfoJSON, _ := json.Marshal(payInfo)

	if err := s.queries.UpdatePaymentInfoRaw(ctx, orderID, tenantID, "pending", payInfoJSON); err != nil {
		return nil, fmt.Errorf("persist VA payment: %w", err)
	}

	slog.Info("VA payment created",
		"external_id", externalID, "bank_code", bankCode, "order_id", orderID)
	return paymentInfoToResponse(&payInfo), nil
}

// HandlePaymentSuccess is the idempotent handler called by Asynq webhook workers.
//
// Flow:
//  1. Find order by external_id (JSONB expression index, no JWT/tenant filter)
//  2. Skip if payment_status is already "paid" (idempotency)
//  3. Atomically deduct stock for every line item via PostgreSQL deduct_stock()
//  4. Mark order paid + processing in a single UPDATE
//  5. Publish SSE event to the tenant's connected browsers
//  6. Enqueue marketplace stock-sync Asynq tasks
func (s *PaymentService) HandlePaymentSuccess(
	ctx context.Context,
	externalID string,
	rawPayload json.RawMessage,
) error {
	order, err := s.queries.GetOrderByExternalPaymentIDRaw(ctx, externalID)
	if err != nil {
		return fmt.Errorf("order lookup external_id=%q: %w", externalID, err)
	}

	// Idempotency guard
	if order.PaymentStatus == string(domain.PaymentStatusPaid) {
		slog.Info("payment already processed, skipping", "external_id", externalID)
		return nil
	}

	var lineItems []domain.LineItem
	if err := json.Unmarshal(order.LineItems, &lineItems); err != nil {
		return fmt.Errorf("parse line_items order %s: %w", order.ID, err)
	}

	// Deduct stock for each line item.
	// deduct_stock() uses SELECT FOR UPDATE inside PostgreSQL, preventing
	// race conditions from concurrent webhook retries.
	for _, item := range lineItems {
		productID, parseErr := uuid.Parse(item.ProductID)
		if parseErr != nil {
			return fmt.Errorf("invalid product_id %q: %w", item.ProductID, parseErr)
		}
		reason := fmt.Sprintf("Payment %s confirmed, order %s", externalID, order.OrderNumber)
		if err := s.queries.DeductStockRaw(ctx,
			order.TenantID, productID, order.ID, int32(item.Quantity), reason,
		); err != nil {
			return fmt.Errorf("deduct_stock product=%s order=%s: %w", item.ProductID, order.ID, err)
		}
	}

	// Update payment_info with paid timestamp
	var payInfo domain.PaymentInfo
	if order.PaymentInfo != nil {
		_ = json.Unmarshal(order.PaymentInfo, &payInfo)
	}
	now := time.Now()
	payInfo.Status = "paid"
	payInfo.PaidAt = &now
	updatedJSON, _ := json.Marshal(payInfo)

	// Single atomic UPDATE for both payment_status and order status
	if err := s.queries.ConfirmPaymentAndProcessOrderRaw(ctx,
		order.ID, order.TenantID,
		string(domain.PaymentStatusPaid),
		string(domain.OrderStatusProcessing),
		updatedJSON,
	); err != nil {
		return fmt.Errorf("confirm payment order %s: %w", order.ID, err)
	}

	// Real-time SSE push to tenant browsers
	s.broker.Publish(order.TenantID.String(), broker.Event{
		Type: "payment.success",
		Data: map[string]any{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       payInfo.Amount,
			"paid_at":      now.Format(time.RFC3339),
		},
	})

	// Enqueue marketplace stock-sync (best-effort; log failures, don't fail the webhook)
	for _, item := range lineItems {
		payload, _ := json.Marshal(worker.StockSyncPayload{
			TenantID:    order.TenantID.String(),
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			NewQuantity: -1,
		})
		task := asynq.NewTask(worker.TypeStockSync, payload)
		if _, enqErr := s.asynq.EnqueueContext(ctx, task,
			asynq.MaxRetry(3), asynq.Queue("stock_sync"),
		); enqErr != nil {
			slog.Warn("stock sync enqueue failed",
				"product_id", item.ProductID, "err", enqErr)
		}
	}

	slog.Info("payment success processed",
		"external_id", externalID,
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"items_deducted", len(lineItems),
	)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func paymentInfoToResponse(p *domain.PaymentInfo) *dto.PaymentResponse {
	r := &dto.PaymentResponse{
		ExternalID:  p.ExternalID,
		PaymentType: string(p.PaymentType),
		Amount:      float64(p.Amount),
		Status:      p.Status,
		QRISString:  p.QRISString,
		VANumber:    p.VANumber,
		BankCode:    p.BankCode,
		VAName:      p.VAName,
	}
	if p.ExpiresAt != nil {
		r.ExpiresAt = p.ExpiresAt.Format(time.RFC3339)
	}
	return r
}

func isValidVABank(code string, supported []string) bool {
	for _, b := range supported {
		if strings.EqualFold(b, code) {
			return true
		}
	}
	return false
}

func sanitizeVAName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 30 {
		return name[:30]
	}
	if name == "" {
		return "Pelanggan"
	}
	return name
}
