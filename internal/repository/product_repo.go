package repository

import (
	"context"
	"fmt"

	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) GetByID(ctx context.Context, tenantID, productID uuid.UUID) (*domain.Product, error) {
	var p domain.Product
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, sku, barcode, name, price, cost_price,
		       stock_quantity, min_stock_level, track_inventory, unit, is_active,
		       created_at, updated_at
		FROM products
		WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE`,
		productID, tenantID,
	).Scan(
		&p.ID, &p.TenantID, &p.SKU, &p.Barcode, &p.Name, &p.Price, &p.CostPrice,
		&p.StockQuantity, &p.MinStockLevel, &p.TrackInventory, &p.Unit, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("product %s: %w", productID, err)
	}
	return &p, nil
}

// GetByIDForUpdate locks the product row with SELECT FOR UPDATE inside an active
// transaction. Any concurrent transaction requesting the same row will block until
// this transaction commits or rolls back — preventing overselling.
func (r *productRepository) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, tenantID, productID uuid.UUID) (*domain.Product, error) {
	var p domain.Product
	err := tx.QueryRow(ctx, `
		SELECT id, tenant_id, sku, name, price,
		       stock_quantity, min_stock_level, track_inventory, unit
		FROM products
		WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE
		FOR UPDATE`,
		productID, tenantID,
	).Scan(
		&p.ID, &p.TenantID, &p.SKU, &p.Name, &p.Price,
		&p.StockQuantity, &p.MinStockLevel, &p.TrackInventory, &p.Unit,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("produk %s tidak ditemukan atau tidak aktif", productID)
		}
		return nil, fmt.Errorf("lock produk %s: %w", productID, err)
	}
	return &p, nil
}

// DeductStock decrements stock within an existing transaction and returns the new quantity.
func (r *productRepository) DeductStock(ctx context.Context, tx pgx.Tx, tenantID, productID uuid.UUID, qty int) (int, error) {
	var newQty int
	err := tx.QueryRow(ctx, `
		UPDATE products
		SET stock_quantity = stock_quantity - $1,
		    updated_at     = NOW()
		WHERE id = $2 AND tenant_id = $3
		RETURNING stock_quantity`,
		qty, productID, tenantID,
	).Scan(&newQty)
	if err != nil {
		return 0, fmt.Errorf("deduct stok produk %s: %w", productID, err)
	}
	return newQty, nil
}
