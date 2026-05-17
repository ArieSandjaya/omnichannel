package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, tx pgx.Tx, o *domain.Order) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO orders (
			id, tenant_id, order_number, channel, status,
			customer_info, line_items,
			subtotal, discount_amount, tax_amount, total_amount,
			payment_method, payment_status,
			notes, created_by, created_at, updated_at
		) VALUES (
			$1,  $2,  $3,  $4,  $5,
			$6,  $7,
			$8,  $9,  $10, $11,
			$12, $13,
			$14, $15, $16, $17
		)`,
		o.ID, o.TenantID, o.OrderNumber, o.Channel, o.Status,
		o.CustomerInfo, o.LineItems,
		o.Subtotal, o.DiscountAmount, o.TaxAmount, o.TotalAmount,
		o.PaymentMethod, o.PaymentStatus,
		o.Notes, o.CreatedBy, o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Order, error) {
	var o domain.Order
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, order_number, channel, status,
		       customer_info, line_items,
		       subtotal, discount_amount, tax_amount, total_amount,
		       payment_method, payment_status,
		       notes, created_by, created_at, updated_at
		FROM orders
		WHERE id = $1 AND tenant_id = $2`,
		orderID, tenantID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &o.Channel, &o.Status,
		&o.CustomerInfo, &o.LineItems,
		&o.Subtotal, &o.DiscountAmount, &o.TaxAmount, &o.TotalAmount,
		&o.PaymentMethod, &o.PaymentStatus,
		&o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("order %s: %w", orderID, err)
	}
	return &o, nil
}

// GenerateOrderNumber produces a unique, human-readable order number.
// Format: {PREFIX}-{YYYYMMDD}-{nanosecond modulo 1_000_000}
func (r *orderRepository) GenerateOrderNumber(_ context.Context, _ uuid.UUID, channel string) (string, error) {
	prefixes := map[string]string{
		"pos":       "POS",
		"website":   "WEB",
		"shopee":    "SPE",
		"tokopedia": "TKP",
		"tiktok":    "TKT",
	}
	prefix, ok := prefixes[channel]
	if !ok {
		prefix = "ORD"
	}
	now := time.Now()
	return fmt.Sprintf("%s-%s-%06d", prefix, now.Format("20060102"), now.UnixNano()%1_000_000), nil
}
