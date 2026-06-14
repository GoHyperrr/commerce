package payments

import (
	"context"
)

// PaymentIntent holds the response payload from the payment gateway setup phase.
type PaymentIntent struct {
	ProviderTransactionID string            `json:"provider_transaction_id"`
	ClientSecret          string            `json:"client_secret,omitempty"` // For front-end SDKs
	Amount                float64           `json:"amount"`
	Currency              string            `json:"currency"`
	Status                string            `json:"status"`
	Metadata              map[string]string `json:"metadata,omitempty"`
}

// PaymentProvider defines the operations that each payment integration must implement.
type PaymentProvider interface {
	ID() string
	
	// CreateIntent registers a payment with the provider.
	CreateIntent(ctx context.Context, orderID string, amount float64, currency string) (*PaymentIntent, error)
	
	// VerifyPayment confirms signatures or tokens from a checkout payload.
	VerifyPayment(ctx context.Context, payload map[string]any) (bool, error)
	
	// Refund processes a refund for a successful payment.
	Refund(ctx context.Context, transactionID string, amount float64) (string, error)
}
