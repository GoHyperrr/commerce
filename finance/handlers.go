package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdempotencyKey prevents duplicate processing of the same operation.
type IdempotencyKey struct {
	ID        string    `gorm:"primaryKey"`
	Scope     string    `gorm:"index:idx_scope_key,unique"`
	Key       string    `gorm:"index:idx_scope_key,unique"`
	CreatedAt time.Time
}

// ProcessPayment processes the payment for an order and creates a Payment record.
func (m *Module) ProcessPayment(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	// Idempotency check
	wfID := getString(data, "_workflow_id")
	if wfID != "" {
		processed, err := isProcessed(ctx, m.repo.db, "finance.process_payment", wfID)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
		if processed {
			slog.Info("Payment already processed for this workflow, skipping", "wf_id", wfID)
			var p Payment
			m.repo.db.WithContext(ctx).Where("order_id = ? AND status = ?", getString(workflowInput, "order_id"), PaymentSuccess).First(&p)
			return map[string]any{"payment": &p}, nil
		}
	}

	// We need the order created in the previous step
	oRaw, ok := data["order.create"]
	if !ok {
		return nil, fmt.Errorf("missing result from order.create step")
	}
	resMap, ok := oRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from order.create")
	}

	orderData := resMap["order"]
	var total float64
	var orderID string

	if oGetter, ok := orderData.(interface {
		GetTotal() float64
		GetOrderID() string
	}); ok {
		total = oGetter.GetTotal()
		orderID = oGetter.GetOrderID()
	} else if oMap, ok := orderData.(map[string]any); ok {
		if t, ok := oMap["total_price"].(float64); ok {
			total = t
		}
		if id, ok := oMap["id"].(string); ok {
			orderID = id
		}
	} else {
		// Fallback decode
		type TempOrder struct {
			TotalPrice float64 `json:"total_price"`
			ID         string  `json:"id"`
		}
		var temp TempOrder
		dataBytes, _ := jsonMarshal(orderData)
		if err := jsonUnmarshal(dataBytes, &temp); err == nil {
			total = temp.TotalPrice
			orderID = temp.ID
		}
	}

	if orderID == "" {
		return nil, fmt.Errorf("missing order ID from order.create result")
	}

	forceFail, _ := workflowInput["fail_payment"].(bool)

	paymentID := "pay_" + uuid.New().String()
	
	p := &Payment{
		ID:      paymentID,
		OrderID: orderID,
		Amount:  total,
		Status:  PaymentPending,
	}
	
	if err := m.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save pending payment: %w", err)
	}

	if forceFail {
		p.Status = PaymentFailed
		_ = m.repo.Save(ctx, p)
		return nil, fmt.Errorf("payment gateway rejected transaction")
	}

	p.Status = PaymentSuccess
	if err := m.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save successful payment: %w", err)
	}

	if wfID != "" {
		_ = markProcessed(ctx, m.repo.db, "finance.process_payment", wfID)
	}

	slog.Info("Payment processed successfully", "payment_id", p.ID, "order_id", orderID, "amount", total)
	
	return map[string]any{"payment": p}, nil
}

// CompensatePayment handles refunding a payment if a subsequent step fails.
func (m *Module) CompensatePayment(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// Idempotency check
	wfID := getString(data, "_workflow_id")
	if wfID != "" {
		processed, err := isProcessed(ctx, m.repo.db, "finance.compensate_payment", wfID)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
		if processed {
			slog.Info("Payment already refunded for this workflow, skipping", "wf_id", wfID)
			return nil, nil
		}
	}

	// Check if payment was processed
	resRaw, ok := data["finance.process_payment"]
	if !ok || resRaw == nil {
		return nil, nil // Nothing to compensate if payment wasn't processed
	}
	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, nil
	}
	p, ok := resMap["payment"].(*Payment)
	if !ok {
		return nil, nil
	}

	p.Status = PaymentRefunded
	if err := m.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	if wfID != "" {
		_ = markProcessed(ctx, m.repo.db, "finance.compensate_payment", wfID)
	}

	slog.Warn("Saga Compensation: Payment refunded", "payment_id", p.ID)
	return map[string]any{"payment": p}, nil
}

// ProcessPaymentStep wraps ProcessPayment to mdk.StepHandler.
func (m *Module) ProcessPaymentStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.ProcessPayment(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// CompensatePaymentStep wraps CompensatePayment to mdk.StepHandler.
func (m *Module) CompensatePaymentStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.CompensatePayment(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

func isProcessed(ctx context.Context, db *gorm.DB, scope, key string) (bool, error) {
	var ik IdempotencyKey
	err := db.WithContext(ctx).Table("idempotency_keys").Where("scope = ? AND key = ?", scope, key).First(&ik).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func markProcessed(ctx context.Context, db *gorm.DB, scope, key string) error {
	ik := &IdempotencyKey{
		ID:        "ik_" + uuid.New().String(),
		Scope:     scope,
		Key:       key,
		CreatedAt: time.Now(),
	}
	return db.WithContext(ctx).Table("idempotency_keys").Create(ik).Error
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
