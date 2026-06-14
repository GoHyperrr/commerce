package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handlers struct {
	db *gorm.DB
	rt mdk.Runtime
}

func NewHandlers(db *gorm.DB, rt mdk.Runtime) *Handlers {
	return &Handlers{
		db: db,
		rt: rt,
	}
}

// CreateIntent registers a payment intent/order with the provider and GORM.
func (h *Handlers) CreateIntent(ctx context.Context, orderID string, providerName string) (*PaymentTransaction, error) {
	// 1. Fetch order details from database
	var order struct {
		ID         string
		TotalPrice float64
	}
	err := h.db.WithContext(ctx).Table("orders").Where("id = ?", orderID).First(&order).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order %s: %w", orderID, err)
	}

	// 2. Resolve provider from registry
	prov, ok := GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("payment provider %s not registered", providerName)
	}

	// Default currency based on gateway constraints (Razorpay default domestic test requires INR)
	currency := "USD"
	if providerName == "razorpay" {
		currency = "INR"
	}

	// 3. Call provider to register intent
	intent, err := prov.CreateIntent(ctx, orderID, order.TotalPrice, currency)
	if err != nil {
		return nil, err
	}

	// 4. Create and save PaymentTransaction GORM record
	tx := &PaymentTransaction{
		ID:                    "tx_" + uuid.New().String(),
		OrderID:               orderID,
		Provider:              providerName,
		ProviderTransactionID: intent.ProviderTransactionID,
		Amount:                order.TotalPrice,
		Currency:              currency,
		Status:                StatusPending,
		ClientSecret:          intent.ClientSecret,
	}

	if intent.Metadata != nil {
		if metaBytes, err := json.Marshal(intent.Metadata); err == nil {
			tx.Metadata = string(metaBytes)
		}
	}

	if err := h.db.WithContext(ctx).Save(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to save payment transaction: %w", err)
	}

	h.rt.Logger().Info("Payment intent created successfully", "order_id", orderID, "provider", providerName, "transaction_id", tx.ID)
	return tx, nil
}

// VerifyPayment confirms checkout signature/token values, updating order and transaction states.
func (h *Handlers) VerifyPayment(ctx context.Context, orderID string, providerName string, payload map[string]any) (*PaymentTransaction, error) {
	// 1. Resolve provider
	prov, ok := GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("payment provider %s not registered", providerName)
	}

	// 2. Retrieve pending transaction
	var tx PaymentTransaction
	err := h.db.WithContext(ctx).Where("order_id = ? AND provider = ? AND status = ?", orderID, providerName, StatusPending).First(&tx).Error
	if err != nil {
		return nil, fmt.Errorf("no pending payment transaction found for order %s: %w", orderID, err)
	}

	// 3. Verify signatures via provider client
	verified, err := prov.VerifyPayment(ctx, payload)
	if err != nil {
		tx.Status = StatusFailed
		_ = h.db.WithContext(ctx).Save(&tx).Error
		return nil, fmt.Errorf("payment verification error: %w", err)
	}

	if !verified {
		tx.Status = StatusFailed
		_ = h.db.WithContext(ctx).Save(&tx).Error
		return nil, fmt.Errorf("payment gateway verification failed")
	}

	// 4. Update GORM transaction and order status to paid
	tx.Status = StatusSucceeded
	if err := h.db.WithContext(ctx).Save(&tx).Error; err != nil {
		return nil, fmt.Errorf("failed to update payment transaction: %w", err)
	}

	// Update order status in orders table to PAID
	err = h.db.WithContext(ctx).Table("orders").Where("id = ?", orderID).Update("status", "PAID").Error
	if err != nil {
		h.rt.Logger().Error("failed to update order status to paid", "order_id", orderID, "error", err)
	}

	// 5. Emit order.paid to Event Fabric
	err = h.rt.Bus().Publish(ctx, mdk.Event{
		ID:        "evt_" + uuid.New().String(),
		Namespace: "commerce.order",
		Type:      "paid",
		Payload: map[string]any{
			"order_id":       orderID,
			"amount":         tx.Amount,
			"provider":       providerName,
			"transaction_id": tx.ID,
		},
		OccurredAt: time.Now(),
	})
	if err != nil {
		h.rt.Logger().Error("failed to publish order paid event", "order_id", orderID, "error", err)
	}

	h.rt.Logger().Info("Payment verified and order status updated to PAID", "order_id", orderID, "transaction_id", tx.ID)
	return &tx, nil
}

// Refund processes refunds (used as Sage rollback compensations).
func (h *Handlers) Refund(ctx context.Context, transactionID string, amount float64) (string, error) {
	// 1. Retrieve transaction details
	var tx PaymentTransaction
	err := h.db.WithContext(ctx).Where("provider_transaction_id = ?", transactionID).First(&tx).Error
	if err != nil {
		return "", fmt.Errorf("transaction not found for refund ID %s: %w", transactionID, err)
	}

	// 2. Resolve provider
	prov, ok := GetProvider(tx.Provider)
	if !ok {
		return "", fmt.Errorf("payment provider %s not registered", tx.Provider)
	}

	// 3. Process refund
	refID, err := prov.Refund(ctx, transactionID, amount)
	if err != nil {
		return "", err
	}

	// 4. Update state
	tx.Status = StatusRefunded
	if err := h.db.WithContext(ctx).Save(&tx).Error; err != nil {
		slog.Error("failed to update transaction state to refunded", "tx_id", tx.ID, "error", err)
	}

	h.rt.Logger().Warn("Payment refunded successfully", "provider_transaction_id", transactionID, "amount", amount, "refund_id", refID)
	return refID, nil
}

// CreateIntentStep wraps CreateIntent to mdk.StepHandler.
func (h *Handlers) CreateIntentStep(sCtx mdk.StepContext) mdk.StepResult {
	orderID, _ := sCtx.Input["order_id"].(string)
	provider, _ := sCtx.Input["provider"].(string)

	if orderID == "" || provider == "" {
		return mdk.StepResult{Err: fmt.Errorf("order_id and provider fields required")}
	}

	tx, err := h.CreateIntent(sCtx.Ctx, orderID, provider)
	if err != nil {
		return mdk.StepResult{Err: err}
	}

	return mdk.StepResult{Output: map[string]any{"transaction": tx}}
}

// VerifyPaymentStep wraps VerifyPayment to mdk.StepHandler.
func (h *Handlers) VerifyPaymentStep(sCtx mdk.StepContext) mdk.StepResult {
	orderID, _ := sCtx.Input["order_id"].(string)
	provider, _ := sCtx.Input["provider"].(string)
	payload, _ := sCtx.Input["payload"].(map[string]any)

	if orderID == "" || provider == "" || payload == nil {
		return mdk.StepResult{Err: fmt.Errorf("order_id, provider, and payload fields required")}
	}

	tx, err := h.VerifyPayment(sCtx.Ctx, orderID, provider, payload)
	if err != nil {
		return mdk.StepResult{Err: err}
	}

	return mdk.StepResult{Output: map[string]any{"transaction": tx}}
}

// RefundStep wraps Refund to mdk.StepHandler.
func (h *Handlers) RefundStep(sCtx mdk.StepContext) mdk.StepResult {
	txID, _ := sCtx.Input["provider_transaction_id"].(string)
	amount, _ := sCtx.Input["amount"].(float64)

	if txID == "" || amount <= 0 {
		return mdk.StepResult{Err: fmt.Errorf("provider_transaction_id and positive amount required")}
	}

	refID, err := h.Refund(sCtx.Ctx, txID, amount)
	if err != nil {
		return mdk.StepResult{Err: err}
	}

	return mdk.StepResult{Output: map[string]any{"refund_id": refID}}
}
