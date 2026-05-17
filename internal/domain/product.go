package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	CategoryID    *uuid.UUID `json:"category_id"`
	SKU           string     `json:"sku"`
	Barcode       string     `json:"barcode"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Price         int64      `json:"price"`      // dalam rupiah
	CostPrice     int64      `json:"cost_price"`
	StockQuantity int        `json:"stock_quantity"`
	MinStockLevel int        `json:"min_stock_level"`
	TrackInventory bool      `json:"track_inventory"`
	Unit          string     `json:"unit"`
	ImageURL      string     `json:"image_url"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (p Product) FormatPrice() string {
	return formatRupiah(p.Price)
}

func (p Product) FormatCostPrice() string {
	return formatRupiah(p.CostPrice)
}

func (p Product) IsLowStock() bool {
	threshold := p.MinStockLevel
	if threshold <= 0 {
		threshold = 5
	}
	return p.StockQuantity <= threshold
}

func (p Product) IsOutOfStock() bool {
	return p.StockQuantity <= 0
}

func (p Product) StockBadgeClass() string {
	if p.IsOutOfStock() {
		return "bg-red-100 text-red-700"
	}
	if p.IsLowStock() {
		return "bg-yellow-100 text-yellow-700"
	}
	return "bg-green-100 text-green-700"
}

func (p Product) ImageOrPlaceholder() string {
	if p.ImageURL != "" {
		return p.ImageURL
	}
	return "/static/img/product-placeholder.svg"
}

func formatRupiah(amount int64) string {
	if amount == 0 {
		return "Rp 0"
	}
	s := fmt.Sprintf("%d", amount)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return "Rp " + result
}
