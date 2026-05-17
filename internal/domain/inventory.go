package domain

import (
	"time"

	"github.com/google/uuid"
)

type InventoryChangeType string

const (
	ChangeTypeSale       InventoryChangeType = "sale"
	ChangeTypePurchase   InventoryChangeType = "purchase"
	ChangeTypeAdjustment InventoryChangeType = "adjustment"
	ChangeTypeReturn     InventoryChangeType = "return"
)

type InventoryLog struct {
	ID             uuid.UUID           `db:"id"              json:"id"`
	TenantID       uuid.UUID           `db:"tenant_id"       json:"tenant_id"`
	ProductID      uuid.UUID           `db:"product_id"      json:"product_id"`
	Type           InventoryChangeType `db:"type"            json:"type"`
	QuantityChange int                 `db:"quantity_change" json:"quantity_change"`
	QuantityBefore int                 `db:"quantity_before" json:"quantity_before"`
	QuantityAfter  int                 `db:"quantity_after"  json:"quantity_after"`
	ReferenceType  *string             `db:"reference_type"  json:"reference_type,omitempty"`
	ReferenceID    *uuid.UUID          `db:"reference_id"    json:"reference_id,omitempty"`
	Channel        *string             `db:"channel"         json:"channel,omitempty"`
	Notes          *string             `db:"notes"           json:"notes,omitempty"`
	CreatedBy      *uuid.UUID          `db:"created_by"      json:"created_by,omitempty"`
	CreatedAt      time.Time           `db:"created_at"      json:"created_at"`
}
