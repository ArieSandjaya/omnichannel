package domain

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
