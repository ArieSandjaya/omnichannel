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
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "pending"
	OrderStatusAwaitingPayment OrderStatus = "awaiting_payment"
	OrderStatusPaid            OrderStatus = "paid"
	OrderStatusProcessing      OrderStatus = "processing"
	OrderStatusShipped         OrderStatus = "shipped"
	OrderStatusDelivered       OrderStatus = "delivered"
	OrderStatusCancelled       OrderStatus = "cancelled"
)

// LineItem is stored inside orders.line_items JSONB array.
type LineItem struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"`   // unit price in IDR
	Subtotal  int64  `json:"subtotal"` // price * quantity
}

// ShippingInfo is stored as JSONB in orders.shipping_info.
// The biteship_order_id and waybill_id fields are indexed for webhook lookup.
type ShippingInfo struct {
	BiteshipOrderID string `json:"biteship_order_id,omitempty"`
	CourierCode     string `json:"courier_code"`
	CourierService  string `json:"courier_service"`
	CourierName     string `json:"courier_name,omitempty"`
	TrackingNumber  string `json:"tracking_number,omitempty"`
	WaybillID       string `json:"waybill_id,omitempty"`
	Status          string `json:"status"`
	EstimatedDays   string `json:"estimated_days,omitempty"`
	Price           int64  `json:"price"`
}
