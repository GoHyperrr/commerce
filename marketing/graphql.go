package marketing

import (
	"context"
	"fmt"

	"github.com/GoHyperrr/commerce/cart"
	"github.com/google/uuid"
)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getCoupon":         m.GetCoupon,
		"getLoyaltyBalance": m.GetLoyaltyBalance,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"applyCouponToCart": m.ApplyCouponToCart,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) ApplyCouponToCart(ctx context.Context, cartID string, couponCode string) (*cart.Cart, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	workflowInput := map[string]any{
		"cart_id":     cartID,
		"coupon_code": couponCode,
		"product_id":  "DISCOUNT_PROMO",
		"quantity":    1.0,
		"price":       0.0,
	}

	execID := "promo_" + uuid.New().String()
	_, err := executor.ExecuteSync(ctx, execID, "marketing.apply_coupon", workflowInput)
	if err != nil {
		return nil, err
	}

	cartModRaw, ok := m.rt.Module("commerce.cart")
	if !ok {
		return nil, fmt.Errorf("cart module not found")
	}
	cartMod, ok := cartModRaw.(*cart.Module)
	if !ok {
		return nil, fmt.Errorf("invalid cart module instance")
	}

	return cartMod.Repo().GetByID(ctx, cartID)
}

func (m *Module) GetCoupon(ctx context.Context, code string) (*Coupon, error) {
	return m.repo.GetCouponByCode(ctx, code)
}

func (m *Module) GetLoyaltyBalance(ctx context.Context, customerID string) (int, error) {
	lp, err := m.repo.GetLoyaltyPointsByCustomerID(ctx, customerID)
	if err != nil {
		return 0, nil // Default balance
	}
	return lp.Balance, nil
}
