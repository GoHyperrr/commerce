package payments

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) ID() string {
	return "mock"
}

func (m *MockProvider) CreateIntent(ctx context.Context, orderID string, amount float64, currency string) (*PaymentIntent, error) {
	txID := "mock_tx_" + uuid.New().String()
	return &PaymentIntent{
		ProviderTransactionID: txID,
		ClientSecret:          "mock_secret_" + uuid.New().String(),
		Amount:                amount,
		Currency:              currency,
		Status:                "requires_payment_method",
		Metadata:              map[string]string{"order_id": orderID},
	}, nil
}

func (m *MockProvider) VerifyPayment(ctx context.Context, payload map[string]any) (bool, error) {
	// For mock payments, we look for a signature equal to "mock_signature_success" or just success field.
	sig, _ := payload["razorpay_signature"].(string)
	if sig == "mock_signature_failed" {
		return false, fmt.Errorf("mock signature verification failed")
	}
	success, ok := payload["success"].(bool)
	if ok && !success {
		return false, nil
	}
	return true, nil
}

func (m *MockProvider) Refund(ctx context.Context, transactionID string, amount float64) (string, error) {
	return "mock_refund_" + uuid.New().String(), nil
}

func init() {
	RegisterProvider(NewMockProvider())
}
