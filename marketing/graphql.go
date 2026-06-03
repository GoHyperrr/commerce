package marketing

import (
	"context"
	"fmt"

	"github.com/GoHyperrr/commerce/cart"
	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

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

func (m *Module) ApplyCouponToCart(ctx context.Context, cartID string, couponCode string) (*model.Cart, error) {
	wf, err := m.deps.Registry.Get("marketing.apply_coupon")
	if err != nil {
		return nil, err
	}

	workflowInput := map[string]any{
		"cart_id":     cartID,
		"coupon_code": couponCode,
		"product_id":  "DISCOUNT_PROMO",
		"quantity":    1.0,
		"price":       0.0,
	}

	execID := "promo_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	cartModRaw, ok := registry.Get("commerce.cart")
	if !ok {
		return nil, fmt.Errorf("cart module not found")
	}
	cartMod, ok := cartModRaw.(*cart.Module)
	if !ok {
		return nil, fmt.Errorf("invalid cart module instance")
	}

	// The cart.add_item handler returns the updated cart under step ID "apply"
	c, ok := results["apply"].(*cart.Cart)
	if !ok {
		// If apply didn't run because validate returned something else, just fetch the cart
		c, err = cartMod.Repo().GetByID(ctx, cartID)
		if err != nil {
			return nil, err
		}
	}

	// Map cart to model.Cart (cart/graphql.go doesn't export mapCartToModel, so we map it here or construct)
	res := &model.Cart{
		ID:         c.ID,
		CustomerID: c.CustomerID,
		Status:     string(c.Status),
	}
	for _, item := range c.Items {
		res.Items = append(res.Items, &model.CartItem{
			ID:        item.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		})
	}

	return res, nil
}

func (m *Module) GetCoupon(ctx context.Context, code string) (*model.Coupon, error) {
	c, err := m.repo.GetCouponByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return mapCouponToModel(c), nil
}

func (m *Module) GetLoyaltyBalance(ctx context.Context, customerID string) (int, error) {
	lp, err := m.repo.GetLoyaltyPointsByCustomerID(ctx, customerID)
	if err != nil {
		return 0, nil // Default balance
	}
	return lp.Balance, nil
}

func mapCouponToModel(c *Coupon) *model.Coupon {
	return &model.Coupon{
		ID:                 c.ID,
		Code:               c.Code,
		DiscountPercentage: c.DiscountPercentage,
		Active:             c.Active,
	}
}
