package cart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
)

func getStringOrPointer(m map[string]any, key string) *string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		if s == "" {
			return nil
		}
		return &s
	}
	if sp, ok := v.(*string); ok {
		if sp == nil || *sp == "" {
			return nil
		}
		return sp
	}
	return nil
}

func getStringVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if sp, ok := v.(*string); ok {
		if sp == nil {
			return ""
		}
		return *sp
	}
	return ""
}

// AddItem handles adding an item to a cart via workflow.
func (m *Module) AddItem(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	cartID, _ := workflowInput["cart_id"].(string)
	productID, _ := workflowInput["product_id"].(string)

	var quantity int
	if q, ok := workflowInput["quantity"].(int); ok {
		quantity = q
	} else if qf, ok := workflowInput["quantity"].(float64); ok {
		quantity = int(qf)
	}

	var price float64
	if p, ok := workflowInput["price"].(float64); ok {
		price = p
	}

	if cartID == "" || productID == "" || quantity <= 0 {
		return nil, fmt.Errorf("invalid or missing input fields")
	}

	c, err := m.repo.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	if c.Status != CartActive {
		return nil, fmt.Errorf("cart is not active")
	}

	// Check if item already exists
	found := false
	for i, item := range c.Items {
		if item.ProductID == productID {
			c.Items[i].Quantity += quantity
			found = true
			break
		}
	}

	if !found {
		c.Items = append(c.Items, CartItem{
			ID:        "ci_" + uuid.New().String(),
			CartID:    cartID,
			ProductID: productID,
			Quantity:  quantity,
			Price:     price,
		})
	}

	if err := m.repo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	return map[string]any{"cart": c}, nil
}

// RemoveItem handles removing an item from a cart.
func (m *Module) RemoveItem(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	cartID, _ := workflowInput["cart_id"].(string)
	itemID, _ := workflowInput["item_id"].(string)

	if err := m.repo.DeleteItem(ctx, cartID, itemID); err != nil {
		return nil, fmt.Errorf("failed to delete item: %w", err)
	}

	c, err := m.repo.GetByID(ctx, cartID)
	if err != nil {
		return nil, err
	}

	return map[string]any{"cart": c}, nil
}

// Checkout handles finalizing the cart.
func (m *Module) Checkout(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	cartID, _ := workflowInput["cart_id"].(string)
	c, err := m.repo.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	if len(c.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Populate checkout parameters if provided
	c.ShippingAddressID = getStringOrPointer(workflowInput, "shipping_address_id")
	c.BillingAddressID = getStringOrPointer(workflowInput, "billing_address_id")
	c.PaymentMethod = getStringVal(workflowInput, "payment_method")
	c.ShippingCarrier = getStringVal(workflowInput, "shipping_carrier")
	c.ShippingMethod = getStringVal(workflowInput, "shipping_method")

	if shippingAddr, ok := workflowInput["shipping_address"]; ok && shippingAddr != nil {
		addrBytes, _ := json.Marshal(shippingAddr)
		addrStr := string(addrBytes)
		c.ShippingAddressJSON = &addrStr
	}
	if billingAddr, ok := workflowInput["billing_address"]; ok && billingAddr != nil {
		addrBytes, _ := json.Marshal(billingAddr)
		addrStr := string(addrBytes)
		c.BillingAddressJSON = &addrStr
	}

	c.Status = CartCompleted
	if err := m.repo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to complete cart: %w", err)
	}

	m.rt.Logger().Info("Cart checkout completed", "cart_id", cartID)
	return true, nil
}

// AddItemStep wraps AddItem to conform to mdk.StepHandler.
func (m *Module) AddItemStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.AddItem(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, ok := res.(map[string]any)
	if !ok {
		return mdk.StepResult{Err: fmt.Errorf("invalid result format from AddItem")}
	}
	return mdk.StepResult{Output: resMap}
}

// RemoveItemStep wraps RemoveItem to conform to mdk.StepHandler.
func (m *Module) RemoveItemStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.RemoveItem(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, ok := res.(map[string]any)
	if !ok {
		return mdk.StepResult{Err: fmt.Errorf("invalid result format from RemoveItem")}
	}
	return mdk.StepResult{Output: resMap}
}

// CheckoutStep wraps Checkout to conform to mdk.StepHandler.
func (m *Module) CheckoutStep(sCtx mdk.StepContext) mdk.StepResult {
	_, err := m.Checkout(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{}
}
