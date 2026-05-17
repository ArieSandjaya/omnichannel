package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type OrderChannel string

const (
	ChannelPOS       OrderChannel = "pos"
	ChannelShopee    OrderChannel = "shopee"
	ChannelTokopedia OrderChannel = "tokopedia"
	ChannelTikTok    OrderChannel = "tiktok"
	ChannelWebsite   OrderChannel = "website"
)

type PaymentMethod string

const (
	PaymentTunai PaymentMethod = "tunai"
	PaymentQRIS  PaymentMethod = "qris"
	PaymentKartu PaymentMethod = "kartu"
	PaymentTransfer PaymentMethod = "transfer"
)

type OrderItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
	Quantity  int       `json:"quantity"`
	Subtotal  int64     `json:"subtotal"`
}

type Order struct {
	ID            uuid.UUID     `json:"id"`
	TenantID      uuid.UUID     `json:"tenant_id"`
	OrderNumber   string        `json:"order_number"`
	Channel       OrderChannel  `json:"channel"`
	Status        OrderStatus   `json:"status"`
	PaymentMethod PaymentMethod `json:"payment_method"`
	PaymentStatus string        `json:"payment_status"`
	LineItems     []OrderItem   `json:"line_items"`
	Subtotal      int64         `json:"subtotal"`
	DiscountAmount int64        `json:"discount_amount"`
	TaxAmount     int64         `json:"tax_amount"`
	TotalAmount   int64         `json:"total_amount"`
	Notes         string        `json:"notes"`
	CreatedBy     *uuid.UUID    `json:"created_by"`
	CompletedAt   *time.Time    `json:"completed_at"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (o Order) FormatTotal() string {
	return formatRupiah(o.TotalAmount)
}

func (o Order) StatusBadgeClass() string {
	switch o.Status {
	case OrderStatusPaid, OrderStatusCompleted:
		return "bg-green-100 text-green-700"
	case OrderStatusPending:
		return "bg-yellow-100 text-yellow-700"
	case OrderStatusCancelled:
		return "bg-red-100 text-red-700"
	case OrderStatusShipped:
		return "bg-blue-100 text-blue-700"
	default:
		return "bg-gray-100 text-gray-700"
	}
}

func (o Order) StatusLabel() string {
	switch o.Status {
	case OrderStatusPending:
		return "Menunggu"
	case OrderStatusPaid:
		return "Dibayar"
	case OrderStatusShipped:
		return "Dikirim"
	case OrderStatusCompleted:
		return "Selesai"
	case OrderStatusCancelled:
		return "Dibatalkan"
	default:
		return string(o.Status)
	}
}

func (o Order) ChannelLabel() string {
	switch o.Channel {
	case ChannelPOS:
		return "POS"
	case ChannelShopee:
		return "Shopee"
	case ChannelTokopedia:
		return "Tokopedia"
	case ChannelTikTok:
		return "TikTok"
	case ChannelWebsite:
		return "Website"
	default:
		return string(o.Channel)
	}
}
