package order

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/commerce/cart"
	"github.com/GoHyperrr/commerce/customer"
	"github.com/google/uuid"
)

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getOrder":           m.GetOrder,
		"listOrders":         m.ListOrders,
		"listCustomerOrders": m.ListCustomerOrders,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"createOrderFromCart": m.CreateOrderFromCart,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) CreateOrderFromCart(ctx context.Context, cartID string, input *cart.CheckoutInput) (*Order, error) {
	cartModRaw, ok := m.rt.Module("commerce.cart")
	if !ok {
		return nil, fmt.Errorf("cart module not found")
	}
	cartMod, ok := cartModRaw.(*cart.Module)
	if !ok {
		return nil, fmt.Errorf("invalid cart module instance")
	}

	c, err := cartMod.Repo().GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	if len(c.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Prepare workflow input
	items := make([]any, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, map[string]any{
			"product_id": item.ProductID,
			"quantity":   float64(item.Quantity), // json/map expects float64 for numbers
			"price":      item.Price,
		})
	}

	shippingAddressID := ""
	billingAddressID := ""
	var shippingAddress *customer.CreateAddressInput
	var billingAddress *customer.CreateAddressInput
	var shippingAddressJSON *string
	var billingAddressJSON *string
	paymentMethod := ""
	var paymentDetails map[string]any
	shippingCarrier := ""
	shippingMethod := ""

	if input != nil {
		if input.ShippingAddressID != nil {
			shippingAddressID = *input.ShippingAddressID
		}
		if input.BillingAddressID != nil {
			billingAddressID = *input.BillingAddressID
		}
		shippingAddress = input.ShippingAddress
		billingAddress = input.BillingAddress
		if input.PaymentMethod != nil {
			paymentMethod = *input.PaymentMethod
		}
		paymentDetails = input.PaymentDetails
		if input.ShippingCarrier != nil {
			shippingCarrier = *input.ShippingCarrier
		}
		if input.ShippingMethod != nil {
			shippingMethod = *input.ShippingMethod
		}
	}

	// Fallback to cart fields if input doesn't provide them
	if shippingAddressID == "" && c.ShippingAddressID != nil {
		shippingAddressID = *c.ShippingAddressID
	}
	if billingAddressID == "" && c.BillingAddressID != nil {
		billingAddressID = *c.BillingAddressID
	}
	if shippingAddress == nil && shippingAddressID == "" && c.ShippingAddressJSON != nil {
		shippingAddressJSON = c.ShippingAddressJSON
	}
	if billingAddress == nil && billingAddressID == "" && c.BillingAddressJSON != nil {
		billingAddressJSON = c.BillingAddressJSON
	}
	if paymentMethod == "" {
		paymentMethod = c.PaymentMethod
	}
	if shippingCarrier == "" {
		shippingCarrier = c.ShippingCarrier
	}
	if shippingMethod == "" {
		shippingMethod = c.ShippingMethod
	}

	workflowInput := map[string]any{
		"customer_id": c.CustomerID,
		"cart_id":     c.ID,
		"items":       items,
	}

	if shippingAddressID != "" {
		workflowInput["shipping_address_id"] = shippingAddressID
	}
	if billingAddressID != "" {
		workflowInput["billing_address_id"] = billingAddressID
	}
	if shippingAddress != nil {
		workflowInput["shipping_address"] = shippingAddress
	} else if shippingAddressJSON != nil {
		workflowInput["shipping_address_json"] = *shippingAddressJSON
	}
	if billingAddress != nil {
		workflowInput["billing_address"] = billingAddress
	} else if billingAddressJSON != nil {
		workflowInput["billing_address_json"] = *billingAddressJSON
	}
	if paymentMethod != "" {
		workflowInput["payment_method"] = paymentMethod
	}
	if paymentDetails != nil {
		workflowInput["payment_details"] = paymentDetails
	}
	if shippingCarrier != "" {
		workflowInput["shipping_carrier"] = shippingCarrier
	}
	if shippingMethod != "" {
		workflowInput["shipping_method"] = shippingMethod
	}

	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	execID := "fulfill_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "fulfillment.v1", workflowInput)
	if err != nil {
		return nil, err
	}

	oRaw, ok := results["order.finalize"]
	if !ok {
		oRaw, ok = results["order.create"]
	}
	if !ok {
		return nil, fmt.Errorf("failed to retrieve order from workflow results")
	}

	resMap, ok := oRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from order step")
	}

	var o Order
	if err := decodeResult(resMap["order"], &o); err != nil {
		return nil, fmt.Errorf("invalid order type in results: %w", err)
	}

	return &o, nil
}

func (m *Module) GetOrder(ctx context.Context, id string) (*Order, error) {
	return m.repo.GetByID(ctx, id)
}

func (m *Module) ListOrders(ctx context.Context) ([]*Order, error) {
	return m.repo.List(ctx)
}

func (m *Module) ListCustomerOrders(ctx context.Context, customerID string) ([]*Order, error) {
	return m.repo.ListByCustomerID(ctx, customerID)
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
