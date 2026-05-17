package dto

type CreateQRISRequest struct {
	CustomerName  string `json:"customer_name"  binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
}

type CreateVARequest struct {
	CustomerName string `json:"customer_name" binding:"required"`
	// BankCode must be one of: BCA, BNI, BRI, MANDIRI, PERMATA, BSI
	BankCode string `json:"bank_code" binding:"required"`
}

type PaymentResponse struct {
	ExternalID   string  `json:"external_id"`
	PaymentType  string  `json:"payment_type"`
	Amount       float64 `json:"amount"`
	Status       string  `json:"status"`
	// QRIS
	QRISString string `json:"qris_string,omitempty"`
	// Virtual Account
	VANumber string `json:"va_number,omitempty"`
	BankCode string `json:"bank_code,omitempty"`
	VAName   string `json:"va_name,omitempty"`
	// Meta
	ExpiresAt string `json:"expires_at,omitempty"`
}
