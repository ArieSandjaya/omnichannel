package domain

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID             uuid.UUID `db:"id"             json:"id"`
	TenantID       uuid.UUID `db:"tenant_id"      json:"tenant_id"`
	SKU            string    `db:"sku"             json:"sku"`
	Barcode        *string   `db:"barcode"         json:"barcode,omitempty"`
	Name           string    `db:"name"            json:"name"`
	Price          float64   `db:"price"           json:"price"`
	CostPrice      *float64  `db:"cost_price"      json:"cost_price,omitempty"`
	StockQuantity  int       `db:"stock_quantity"  json:"stock_quantity"`
	MinStockLevel  int       `db:"min_stock_level" json:"min_stock_level"`
	TrackInventory bool      `db:"track_inventory" json:"track_inventory"`
	Unit           string    `db:"unit"            json:"unit"`
	IsActive       bool      `db:"is_active"       json:"is_active"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"      json:"updated_at"`
}
