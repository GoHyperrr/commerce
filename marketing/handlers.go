package marketing

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

// ValidateCoupon checks if a coupon is valid and returns it.
func (m *Module) ValidateCoupon(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	code, _ := workflowInput["coupon_code"].(string)
	if code == "" {
		return nil, fmt.Errorf("coupon code is required")
	}

	coupon, err := m.repo.GetCouponByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("invalid or inactive coupon")
	}

	slog.Info("Coupon validated", "code", code, "discount", coupon.DiscountPercentage)
	return map[string]any{"coupon": coupon}, nil
}

// AddLoyaltyPoints adds points based on the order total.
func (m *Module) AddLoyaltyPoints(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// We need the order created in the previous step
	oRaw, ok := data["order.finalize"]
	if !ok {
		// Fallback to order.create if finalize didn't happen yet
		oRaw, ok = data["order.create"]
	}
	if !ok {
		return nil, fmt.Errorf("missing order from previous step")
	}
	
	resMap, ok := oRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from order step")
	}

	orderData := resMap["order"]
	var total float64
	var customerID string

	if oGetter, ok := orderData.(interface {
		GetTotal() float64
		GetCustomerID() string
	}); ok {
		total = oGetter.GetTotal()
		customerID = oGetter.GetCustomerID()
	} else if oMap, ok := orderData.(map[string]any); ok {
		if t, ok := oMap["total_price"].(float64); ok {
			total = t
		}
		if c, ok := oMap["customer_id"].(string); ok {
			customerID = c
		}
	} else {
		// Fallback decode
		type TempOrder struct {
			TotalPrice float64 `json:"total_price"`
			CustomerID string  `json:"customer_id"`
		}
		var temp TempOrder
		dataBytes, _ := jsonMarshal(orderData)
		if err := jsonUnmarshal(dataBytes, &temp); err == nil {
			total = temp.TotalPrice
			customerID = temp.CustomerID
		}
	}

	if customerID == "" {
		return nil, fmt.Errorf("missing customer ID from order result")
	}

	// Calculate points: 1 point per 10 currency units
	pointsToAdd := int(total / 10)

	lp, err := m.repo.GetLoyaltyPointsByCustomerID(ctx, customerID)
	if err != nil {
		// Auto-create loyalty account if it doesn't exist
		lp = &LoyaltyPoints{
			ID:         "lp_" + uuid.New().String(),
			CustomerID: customerID,
			Balance:    0,
		}
	}

	// Idempotency check
	wfID := getString(data, "_workflow_id")
	if wfID != "" {
		processed, err := isProcessed(ctx, m.repo.db, "marketing.add_loyalty_points", wfID)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
		if processed {
			slog.Info("Loyalty points already added for this workflow, skipping", "wf_id", wfID)
			return map[string]any{"loyalty_points": lp}, nil
		}
	}

	lp.Balance += pointsToAdd
	if err := m.repo.SaveLoyaltyPoints(ctx, lp); err != nil {
		return nil, err
	}

	if wfID != "" {
		_ = markProcessed(ctx, m.repo.db, "marketing.add_loyalty_points", wfID)
	}

	slog.Info("Loyalty points added", "customer_id", customerID, "points", pointsToAdd, "new_balance", lp.Balance)
	return map[string]any{"loyalty_points": lp}, nil
}

// ValidateCouponStep wraps ValidateCoupon to mdk.StepHandler.
func (m *Module) ValidateCouponStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.ValidateCoupon(sCtx.Ctx, map[string]any{
		"input": sCtx.Input,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// AddLoyaltyPointsStep wraps AddLoyaltyPoints to mdk.StepHandler.
func (m *Module) AddLoyaltyPointsStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.AddLoyaltyPoints(sCtx.Ctx, sCtx.Input)
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

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
