package payments

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getPaymentTransaction":      m.GetPaymentTransaction,
		"listOrderTransactions":      m.ListOrderTransactions,
		"listActivePaymentProviders": m.ListActivePaymentProviders,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"createPaymentIntent": m.CreatePaymentIntent,
		"verifyPayment":       m.VerifyPayment,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

type PaymentProviderInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type PaymentVerificationResult struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (m *Module) GetPaymentTransaction(ctx context.Context, id string) (*PaymentTransaction, error) {
	var tx PaymentTransaction
	err := m.rt.DB().WithContext(ctx).First(&tx, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("transaction %s not found: %w", id, err)
	}
	return &tx, nil
}

func (m *Module) ListOrderTransactions(ctx context.Context, orderID string) ([]*PaymentTransaction, error) {
	var txs []*PaymentTransaction
	err := m.rt.DB().WithContext(ctx).Where("order_id = ?", orderID).Find(&txs).Error
	if err != nil {
		return nil, err
	}
	return txs, nil
}

func (m *Module) ListActivePaymentProviders(ctx context.Context) ([]*PaymentProviderInfo, error) {
	var list []*PaymentProviderInfo

	// Always add the mock provider
	list = append(list, &PaymentProviderInfo{
		ID:      "mock",
		Name:    "Developer Mock Gateway",
		Enabled: true,
	})

	// Check Stripe
	stripeKey, _ := m.rt.Config("stripe_secret_key").(string)
	list = append(list, &PaymentProviderInfo{
		ID:      "stripe",
		Name:    "Stripe Elements",
		Enabled: stripeKey != "",
	})

	// Check Razorpay
	razorpayKey, _ := m.rt.Config("razorpay_key_secret").(string)
	list = append(list, &PaymentProviderInfo{
		ID:      "razorpay",
		Name:    "Razorpay Custom Checkout",
		Enabled: razorpayKey != "",
	})

	return list, nil
}

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) CreatePaymentIntent(ctx context.Context, orderID string, provider string) (*PaymentIntent, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	execID := "pay_intent_" + uuid.New().String()
	input := map[string]any{
		"order_id": orderID,
		"provider": provider,
	}

	results, err := executor.ExecuteSync(ctx, execID, "payments.create_intent", input)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["payments.create_intent"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve payment intent result from workflow")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from create intent step")
	}

	txRaw, ok := resMap["transaction"]
	if !ok {
		return nil, fmt.Errorf("missing transaction in create intent result")
	}

	txBytes, err := json.Marshal(txRaw)
	if err != nil {
		return nil, err
	}

	var tx PaymentTransaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return nil, err
	}

	return &PaymentIntent{
		ProviderTransactionID: tx.ProviderTransactionID,
		ClientSecret:          tx.ClientSecret,
		Amount:                tx.Amount,
		Currency:              tx.Currency,
		Status:                string(tx.Status),
	}, nil
}

func (m *Module) VerifyPayment(ctx context.Context, orderID string, provider string, payload map[string]any) (*PaymentVerificationResult, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	execID := "pay_verify_" + uuid.New().String()
	input := map[string]any{
		"order_id": orderID,
		"provider": provider,
		"payload":  payload,
	}

	results, err := executor.ExecuteSync(ctx, execID, "payments.verify", input)
	if err != nil {
		return &PaymentVerificationResult{
			OrderID: orderID,
			Status:  "FAILED",
			Message: err.Error(),
		}, nil
	}

	resRaw, ok := results["payments.verify"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve verification result from workflow")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from verify step")
	}

	txRaw, ok := resMap["transaction"]
	if !ok {
		return nil, fmt.Errorf("missing transaction in verify result")
	}

	txBytes, err := json.Marshal(txRaw)
	if err != nil {
		return nil, err
	}

	var tx PaymentTransaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return nil, err
	}

	statusStr := string(tx.Status)
	msg := "Payment verified successfully"
	if tx.Status != StatusSucceeded {
		msg = "Payment verification failed"
	}

	return &PaymentVerificationResult{
		OrderID: orderID,
		Status:  statusStr,
		Message: msg,
	}, nil
}
