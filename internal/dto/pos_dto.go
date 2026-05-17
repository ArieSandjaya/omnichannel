package dto

import "github.com/google/uuid"

type POSItem struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity"   validate:"required,min=1"`
	Discount  float64   `json:"discount"   validate:"min=0"`
}

type POSTransactionRequest struct {
	Items         []POSItem `json:"items"          validate:"required,min=1,dive"`
	PaymentMethod string    `json:"payment_method" validate:"required,oneof=cash qris card transfer"`
	AmountPaid    float64   `json:"amount_paid"    validate:"min=0"`
	CustomerName  string    `json:"customer_name"`
	Notes         string    `json:"notes"`
}

type POSItemResult struct {
	ProductID  uuid.UUID `json:"product_id"`
	Name       string    `json:"name"`
	Quantity   int       `json:"quantity"`
	UnitPrice  float64   `json:"unit_price"`
	Subtotal   float64   `json:"subtotal"`
	StockAfter int       `json:"stock_after"`
}

type POSTransactionResponse struct {
	OrderID       uuid.UUID      `json:"order_id"`
	OrderNumber   string         `json:"order_number"`
	TotalAmount   float64        `json:"total_amount"`
	AmountPaid    float64        `json:"amount_paid"`
	Change        float64        `json:"change"`
	PaymentStatus string         `json:"payment_status"`
	Items         []POSItemResult `json:"items"`
}
