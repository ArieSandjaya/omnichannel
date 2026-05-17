package repository

import (
	"context"

	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProductRepository handles product data access.
type ProductRepository interface {
	// GetByID fetches a product without locking (read-only).
	GetByID(ctx context.Context, tenantID, productID uuid.UUID) (*domain.Product, error)

	// GetByIDForUpdate fetches and exclusively locks the product row within tx.
	// Must be called inside an active transaction to prevent concurrent stock deductions.
	GetByIDForUpdate(ctx context.Context, tx pgx.Tx, tenantID, productID uuid.UUID) (*domain.Product, error)

	// DeductStock subtracts qty from stock_quantity within tx and returns the new quantity.
	DeductStock(ctx context.Context, tx pgx.Tx, tenantID, productID uuid.UUID, qty int) (int, error)
}

// OrderRepository handles order persistence.
type OrderRepository interface {
	Create(ctx context.Context, tx pgx.Tx, order *domain.Order) error
	GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Order, error)
	GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID, channel string) (string, error)
}

// InventoryRepository handles immutable stock-change audit logs.
type InventoryRepository interface {
	CreateLog(ctx context.Context, tx pgx.Tx, log *domain.InventoryLog) error
}
