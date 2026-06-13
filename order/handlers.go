package order

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoHyperrr/commerce/customer"
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

// CreateOrder initializes the order in PENDING state.
func (m *Module) CreateOrder(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	customerID, _ := workflowInput["customer_id"].(string)
	cartID, _ := workflowInput["cart_id"].(string)
	itemsRaw, ok := workflowInput["items"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing items in input")
	}

	if customerID == "" || cartID == "" || len(itemsRaw) == 0 {
		return nil, fmt.Errorf("invalid or missing input fields")
	}

	orderID := "ord_" + uuid.New().String()
	o := &Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     StatusPending,
	}

	// Address and payment/shipping parameters propagation
	o.ShippingAddressID = getStringOrPointer(workflowInput, "shipping_address_id")
	o.BillingAddressID = getStringOrPointer(workflowInput, "billing_address_id")
	o.PaymentMethod = getStringVal(workflowInput, "payment_method")
	o.ShippingCarrier = getStringVal(workflowInput, "shipping_carrier")
	o.ShippingMethod = getStringVal(workflowInput, "shipping_method")

	// Resolve Shipping Address JSON Snapshot
	if o.ShippingAddressID != nil {
		var addr customer.Address
		if err := m.rt.DB().WithContext(ctx).First(&addr, "id = ?", *o.ShippingAddressID).Error; err == nil {
			addrBytes, _ := json.Marshal(addr)
			addrStr := string(addrBytes)
			o.ShippingAddressJSON = &addrStr
		}
	} else if shippingAddr, ok := workflowInput["shipping_address"]; ok && shippingAddr != nil {
		addrBytes, _ := json.Marshal(shippingAddr)
		addrStr := string(addrBytes)
		o.ShippingAddressJSON = &addrStr
	} else if shippingAddrJSON := getStringOrPointer(workflowInput, "shipping_address_json"); shippingAddrJSON != nil {
		o.ShippingAddressJSON = shippingAddrJSON
	}

	// Resolve Billing Address JSON Snapshot
	if o.BillingAddressID != nil {
		var addr customer.Address
		if err := m.rt.DB().WithContext(ctx).First(&addr, "id = ?", *o.BillingAddressID).Error; err == nil {
			addrBytes, _ := json.Marshal(addr)
			addrStr := string(addrBytes)
			o.BillingAddressJSON = &addrStr
		}
	} else if billingAddr, ok := workflowInput["billing_address"]; ok && billingAddr != nil {
		addrBytes, _ := json.Marshal(billingAddr)
		addrStr := string(addrBytes)
		o.BillingAddressJSON = &addrStr
	} else if billingAddrJSON := getStringOrPointer(workflowInput, "billing_address_json"); billingAddrJSON != nil {
		o.BillingAddressJSON = billingAddrJSON
	}

	var totalPrice float64
	for _, itemRaw := range itemsRaw {
		itemMap, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}

		var quantity int
		if q, ok := itemMap["quantity"].(int); ok {
			quantity = q
		} else if qf, ok := itemMap["quantity"].(float64); ok {
			quantity = int(qf)
		}

		var price float64
		if p, ok := itemMap["price"].(float64); ok {
			price = p
		} else if pi, ok := itemMap["price"].(int); ok {
			price = float64(pi)
		}

		productID, _ := itemMap["product_id"].(string)
		if productID == "" || quantity <= 0 {
			continue
		}
		
		o.Items = append(o.Items, OrderItem{
			ID:        "oi_" + uuid.New().String() + "_" + productID,
			OrderID:   orderID,
			ProductID: productID,
			Quantity:  quantity,
			UnitPrice: price,
		})
		totalPrice += price * float64(quantity)
	}
	o.TotalPrice = totalPrice

	if err := m.repo.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Emit domain event for observability
	if m.bus != nil {
		wfID, _ := data["_workflow_id"].(string)
		_ = m.bus.Publish(ctx, mdk.Event{
			ID:        "evt_ord_cre_" + uuid.New().String(),
			Namespace: "commerce.order",
			Type:      EventOrderCreated,
			TraceID:   wfID,
			Payload: map[string]any{
				"id":          wfID, // For Projector to correlate with lineage
				"order_id":    orderID,
				"total_price": totalPrice,
				"customer_id": customerID,
			},
			OccurredAt: time.Now(),
		})
	}

	m.rt.Logger().Info("Order created (Pending)", "order_id", orderID, "cart_id", cartID)
	return map[string]any{"order": o}, nil
}

// FinalizeOrder updates status to PAID.
func (m *Module) FinalizeOrder(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	oRaw, ok := data["order.create"]
	if !ok || oRaw == nil {
		return nil, fmt.Errorf("missing result from order.create step")
	}
	resMap, ok := oRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from order.create")
	}
	o, ok := resMap["order"].(*Order)
	if !ok {
		return nil, fmt.Errorf("invalid order type in order.create result")
	}

	o.Status = StatusPaid
	if err := m.repo.Save(ctx, o); err != nil {
		return nil, err
	}

	// Emit domain event
	if m.bus != nil {
		wfID, _ := data["_workflow_id"].(string)
		_ = m.bus.Publish(ctx, mdk.Event{
			ID:        "evt_ord_paid_" + uuid.New().String(),
			Namespace: "commerce.order",
			Type:      EventOrderPaid,
			TraceID:   wfID,
			Payload: map[string]any{
				"id":          wfID,
				"order_id":    o.ID,
				"total_price": o.TotalPrice,
				"customer_id": o.CustomerID,
			},
			OccurredAt: time.Now(),
		})
	}

	m.rt.Logger().Info("Order finalized (Paid)", "order_id", o.ID)
	return map[string]any{"order": o}, nil
}

// CompensatePayment handles payment failure by cancelling the order.
func (m *Module) CompensatePayment(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// Check if order was created
	oRaw, ok := data["order.create"]
	if !ok || oRaw == nil {
		return nil, nil // Nothing to compensate if order wasn't even created
	}
	resMap, ok := oRaw.(map[string]any)
	if !ok {
		return nil, nil
	}
	var o Order
	if err := decodeResult(resMap["order"], &o); err != nil {
		return nil, nil
	}

	o.Status = StatusCancelled
	if err := m.repo.Save(ctx, &o); err != nil {
		return nil, err
	}

	m.rt.Logger().Warn("Saga Compensation: Order cancelled due to payment failure", "order_id", o.ID)
	return nil, nil
}


// CreateOrderStep wraps CreateOrder to mdk.StepHandler.
func (m *Module) CreateOrderStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.CreateOrder(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, ok := res.(map[string]any)
	if !ok {
		return mdk.StepResult{Err: fmt.Errorf("invalid result format from CreateOrder")}
	}
	return mdk.StepResult{Output: resMap}
}

// FinalizeOrderStep wraps FinalizeOrder to mdk.StepHandler.
func (m *Module) FinalizeOrderStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.FinalizeOrder(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, ok := res.(map[string]any)
	if !ok {
		return mdk.StepResult{Err: fmt.Errorf("invalid result format from FinalizeOrder")}
	}
	return mdk.StepResult{Output: resMap}
}

// CompensatePaymentStep wraps CompensatePayment to mdk.StepHandler.
func (m *Module) CompensatePaymentStep(sCtx mdk.StepContext) mdk.StepResult {
	_, err := m.CompensatePayment(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{}
}
