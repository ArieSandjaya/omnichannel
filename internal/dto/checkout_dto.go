package dto

import "github.com/google/uuid"

type CheckoutItem struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity"   validate:"required,min=1"`
	Discount  float64   `json:"discount"   validate:"min=0"`
}

type CustomerInfo struct {
	Name    string `json:"name"  validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

type CheckoutRequest struct {
	Items         []CheckoutItem `json:"items"          validate:"required,min=1,dive"`
	Customer      CustomerInfo   `json:"customer"       validate:"required"`
	PaymentMethod string         `json:"payment_method" validate:"required,oneof=qris transfer cod"`
	Notes         string         `json:"notes"`
}

type CheckoutResponse struct {
	OrderID       uuid.UUID `json:"order_id"`
	OrderNumber   string    `json:"order_number"`
	Subtotal      float64   `json:"subtotal"`
	TaxAmount     float64   `json:"tax_amount"`
	TotalAmount   float64   `json:"total_amount"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
}
