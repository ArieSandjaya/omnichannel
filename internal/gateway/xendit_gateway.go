// Package gateway wraps external payment and shipping SDKs behind interfaces.
// This keeps service layer tests independent of third-party SDK implementations.
package gateway

import (
	"context"
	"fmt"
	"time"

	xendit "github.com/xendit/xendit-go"
	"github.com/xendit/xendit-go/qrcode"
	"github.com/xendit/xendit-go/virtualaccount"
)

// QRISResponse is the simplified response from a QRIS creation call.
type QRISResponse struct {
	XenditID string
	QRString string
}

// VAResponse is the simplified response from a Virtual Account creation call.
type VAResponse struct {
	XenditID      string
	AccountNumber string
}

// XenditGateway abstracts the Xendit SDK calls used by PaymentService.
type XenditGateway interface {
	CreateQRIS(ctx context.Context, externalID, callbackURL string, amount float64) (*QRISResponse, error)
	CreateFixedVA(ctx context.Context, externalID, bankCode, name string, amount float64, expiresAt time.Time) (*VAResponse, error)
}

// xenditGatewayImpl is the production implementation that calls the real Xendit SDK.
type xenditGatewayImpl struct {
	secretKey string
}

// NewXenditGateway creates a production Xendit gateway with the given secret key.
func NewXenditGateway(secretKey string) XenditGateway {
	xendit.Opt.SecretKey = secretKey
	return &xenditGatewayImpl{secretKey: secretKey}
}

func (g *xenditGatewayImpl) CreateQRIS(ctx context.Context, externalID, callbackURL string, amount float64) (*QRISResponse, error) {
	resp, err := qrcode.CreateQRCode(&qrcode.CreateQRCodeParams{
		ExternalID:  externalID,
		Type:        xendit.DynamicQRCode,
		CallbackURL: callbackURL,
		Amount:      amount,
	})
	if err != nil {
		return nil, fmt.Errorf("xendit CreateQRCode: %w", err)
	}

	return &QRISResponse{
		XenditID: resp.ID,
		QRString: resp.QRString,
	}, nil
}

func (g *xenditGatewayImpl) CreateFixedVA(ctx context.Context, externalID, bankCode, name string, amount float64, expiresAt time.Time) (*VAResponse, error) {
	trueVal := true
	resp, err := virtualaccount.CreateFixedVA(&virtualaccount.CreateFixedVAParams{
		ExternalID:     externalID,
		BankCode:       bankCode,
		Name:           name,
		ExpectedAmount: amount,
		IsClosed:       &trueVal,
		IsSingleUse:    &trueVal,
		ExpirationDate: &expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("xendit CreateFixedVA: %w", err)
	}

	return &VAResponse{
		XenditID:      resp.ID,
		AccountNumber: resp.AccountNumber,
	}, nil
}
