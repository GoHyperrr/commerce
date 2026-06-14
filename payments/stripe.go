package payments

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/refund"
)

type StripeProvider struct {
	secretKey string
}

func NewStripeProvider(secretKey string) *StripeProvider {
	return &StripeProvider{secretKey: secretKey}
}

func (s *StripeProvider) ID() string {
	return "stripe"
}

func (s *StripeProvider) CreateIntent(ctx context.Context, orderID string, amount float64, currency string) (*PaymentIntent, error) {
	// Stripe expects the amount in the smallest currency unit (e.g. cents)
	cents := int64(amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(cents),
		Currency: stripe.String(currency),
		Metadata: map[string]string{
			"order_id": orderID,
		},
	}

	stripe.Key = s.secretKey

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe payment intent: %w", err)
	}

	return &PaymentIntent{
		ProviderTransactionID: pi.ID,
		ClientSecret:          pi.ClientSecret,
		Amount:                amount,
		Currency:              currency,
		Status:                string(pi.Status),
		Metadata:              pi.Metadata,
	}, nil
}

func (s *StripeProvider) VerifyPayment(ctx context.Context, payload map[string]any) (bool, error) {
	piID, _ := payload["payment_intent_id"].(string)
	if piID == "" {
		return false, fmt.Errorf("missing payment_intent_id in verification payload")
	}

	stripe.Key = s.secretKey
	pi, err := paymentintent.Get(piID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve Stripe payment intent: %w", err)
	}

	return pi.Status == stripe.PaymentIntentStatusSucceeded, nil
}

func (s *StripeProvider) Refund(ctx context.Context, transactionID string, amount float64) (string, error) {
	cents := int64(amount * 100)
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(transactionID),
		Amount:        stripe.Int64(cents),
	}

	stripe.Key = s.secretKey
	ref, err := refund.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to process Stripe refund: %w", err)
	}

	return ref.ID, nil
}
