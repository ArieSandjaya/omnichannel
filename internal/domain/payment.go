package domain

import (
	"time"

	"github.com/google/uuid"
)

type PaymentType string

const (
	PaymentTypeQRIS           PaymentType = "qris"
	PaymentTypeVirtualAccount PaymentType = "virtual_account"
)

type PaymentStatus string

const (
	PaymentStatusUnpaid  PaymentStatus = "unpaid"
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusExpired PaymentStatus = "expired"
	PaymentStatusFailed  PaymentStatus = "failed"
)

// PaymentInfo is stored as JSONB in orders.payment_info.
// The external_id field is indexed for fast webhook lookup.
type PaymentInfo struct {
	ExternalID  string      `json:"external_id"`
	PaymentType PaymentType `json:"payment_type"`
	XenditID    string      `json:"xendit_id,omitempty"`
	Amount      int64       `json:"amount"`
	Status      string      `json:"status"`
	// QRIS fields
	QRISString string `json:"qris_string,omitempty"`
	// Virtual Account fields
	VANumber string `json:"va_number,omitempty"`
	BankCode string `json:"bank_code,omitempty"`
	VAName   string `json:"va_name,omitempty"`
	// Timestamps
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
	// Tenant reference (embedded for webhook lookup without JWT)
	TenantID uuid.UUID `json:"tenant_id"`
}

// SupportedVABanks lists the bank codes accepted by Xendit VA.
var SupportedVABanks = []string{"BCA", "BNI", "BRI", "MANDIRI", "PERMATA", "BSI"}
