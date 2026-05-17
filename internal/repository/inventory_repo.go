package repository

import (
	"context"
	"fmt"

	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type inventoryRepository struct {
	db *pgxpool.Pool
}

func NewInventoryRepository(db *pgxpool.Pool) InventoryRepository {
	return &inventoryRepository{db: db}
}

// CreateLog writes an immutable inventory audit entry inside an active transaction.
func (r *inventoryRepository) CreateLog(ctx context.Context, tx pgx.Tx, l *domain.InventoryLog) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO inventory_log (
			id, tenant_id, product_id, type,
			quantity_change, quantity_before, quantity_after,
			reference_type, reference_id, channel,
			notes, created_by, created_at
		) VALUES (
			$1,  $2,  $3,  $4,
			$5,  $6,  $7,
			$8,  $9,  $10,
			$11, $12, $13
		)`,
		l.ID, l.TenantID, l.ProductID, l.Type,
		l.QuantityChange, l.QuantityBefore, l.QuantityAfter,
		l.ReferenceType, l.ReferenceID, l.Channel,
		l.Notes, l.CreatedBy, l.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create inventory log: %w", err)
	}
	return nil
}
