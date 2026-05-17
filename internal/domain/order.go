package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderStatus    string
type PaymentStatus  string
type OrderChannel   string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"

	PaymentStatusUnpaid   PaymentStatus = "unpaid"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusRefunded PaymentStatus = "refunded"

	ChannelPOS      OrderChannel = "pos"
	ChannelWebsite  OrderChannel = "website"
	ChannelShopee   OrderChannel = "shopee"
	ChannelTokopedia OrderChannel = "tokopedia"
)

type LineItem struct {
	ProductID uuid.UUID `json:"product_id"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Discount  float64   `json:"discount"`
	Subtotal  float64   `json:"subtotal"`
}

type Order struct {
	ID             uuid.UUID       `db:"id"              json:"id"`
	TenantID       uuid.UUID       `db:"tenant_id"       json:"tenant_id"`
	OrderNumber    string          `db:"order_number"    json:"order_number"`
	Channel        OrderChannel    `db:"channel"         json:"channel"`
	Status         OrderStatus     `db:"status"          json:"status"`
	CustomerInfo   json.RawMessage `db:"customer_info"   json:"customer_info"`
	LineItems      json.RawMessage `db:"line_items"      json:"line_items"`
	Subtotal       float64         `db:"subtotal"        json:"subtotal"`
	DiscountAmount float64         `db:"discount_amount" json:"discount_amount"`
	TaxAmount      float64         `db:"tax_amount"      json:"tax_amount"`
	TotalAmount    float64         `db:"total_amount"    json:"total_amount"`
	PaymentMethod  *string         `db:"payment_method"  json:"payment_method,omitempty"`
	PaymentStatus  PaymentStatus   `db:"payment_status"  json:"payment_status"`
	Notes          *string         `db:"notes"           json:"notes,omitempty"`
	CreatedBy      *uuid.UUID      `db:"created_by"      json:"created_by,omitempty"`
	CreatedAt      time.Time       `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"      json:"updated_at"`
}
