package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/razorpay/razorpay-go"
)

type RazorpayProvider struct {
	keyID      string
	keySecret  string
	BaseURL    string       // Custom BaseURL for testing
	HTTPClient *http.Client // Custom HTTP client for testing
}

func NewRazorpayProvider(keyID, keySecret string) *RazorpayProvider {
	return &RazorpayProvider{
		keyID:     keyID,
		keySecret: keySecret,
	}
}

func (r *RazorpayProvider) ID() string {
	return "razorpay"
}

func (r *RazorpayProvider) client() *razorpay.Client {
	c := razorpay.NewClient(r.keyID, r.keySecret)
	if r.BaseURL != "" {
		c.BaseURL = r.BaseURL
	}
	if r.HTTPClient != nil {
		c.HTTPClient = r.HTTPClient
	}
	return c
}

func (r *RazorpayProvider) CreateIntent(ctx context.Context, orderID string, amount float64, currency string) (*PaymentIntent, error) {
	client := r.client()

	// Razorpay expects amount in paise (smallest currency unit)
	paise := int(amount * 100)

	params := map[string]interface{}{
		"amount":   paise,
		"currency": currency,
		"receipt":  orderID,
	}

	body, err := client.Order.Create(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Razorpay order: %w", err)
	}

	razorpayOrderID, _ := body["id"].(string)
	if razorpayOrderID == "" {
		return nil, fmt.Errorf("razorpay order creation did not return order id")
	}

	return &PaymentIntent{
		ProviderTransactionID: razorpayOrderID,
		ClientSecret:          razorpayOrderID, // Frontend Razorpay checkout needs the order_id
		Amount:                amount,
		Currency:              currency,
		Status:                "created",
	}, nil
}

func (r *RazorpayProvider) VerifyPayment(ctx context.Context, payload map[string]any) (bool, error) {
	orderID, _ := payload["razorpay_order_id"].(string)
	paymentID, _ := payload["razorpay_payment_id"].(string)
	signature, _ := payload["razorpay_signature"].(string)

	if orderID == "" || paymentID == "" || signature == "" {
		return false, fmt.Errorf("missing signature verification parameters (order_id, payment_id, or signature)")
	}

	// Verify signature: HMAC-SHA256(order_id + "|" + payment_id, secret)
	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(r.keySecret))
	_, err := r.WriteToHash(h, data)
	if err != nil {
		return false, err
	}

	generatedSignature := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(generatedSignature), []byte(signature)), nil
}

func (r *RazorpayProvider) WriteToHash(h interface {
	Write([]byte) (int, error)
}, data string) (int, error) {
	return h.Write([]byte(data))
}

func (r *RazorpayProvider) Refund(ctx context.Context, transactionID string, amount float64) (string, error) {
	client := r.client()

	paise := int(amount * 100)

	// Refund Razorpay payment (paymentID, amount, bodyMap, extraHeadersMap)
	body, err := client.Payment.Refund(transactionID, paise, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to process Razorpay refund: %w", err)
	}

	refundID, _ := body["id"].(string)
	return refundID, nil
}
