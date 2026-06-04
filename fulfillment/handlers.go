package fulfillment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
)

func (m *Module) ReserveInventory(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	itemsRaw, ok := workflowInput["items"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing items in input")
	}

	// We will reserve all items or fail.
	var reservedItems []string

	for _, itemRaw := range itemsRaw {
		itemMap, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}
		productID, _ := itemMap["product_id"].(string)
		
		var quantity int
		if q, ok := itemMap["quantity"].(int); ok {
			quantity = q
		} else if qf, ok := itemMap["quantity"].(float64); ok {
			quantity = int(qf)
		}

		if productID == "" || quantity <= 0 {
			continue
		}

		inv, err := m.repo.GetInventoryByProductID(ctx, productID)
		if err != nil {
			// In MVP, we might auto-create inventory if it doesn't exist just for testing
			inv = &Inventory{
				ID: "inv_" + uuid.New().String(),
				ProductID:         productID,
				AvailableQuantity: 100, // Mock stock
			}
		}

		if inv.AvailableQuantity < quantity {
			return nil, fmt.Errorf("insufficient inventory for product %s", productID)
		}

		inv.AvailableQuantity -= quantity
		if err := m.repo.SaveInventory(ctx, inv); err != nil {
			return nil, fmt.Errorf("failed to update inventory: %w", err)
		}
		reservedItems = append(reservedItems, productID)
	}

	slog.Info("Inventory reserved successfully", "items", reservedItems)
	return map[string]any{"reserved": true, "items": itemsRaw}, nil
}

func (m *Module) ReleaseInventory(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// Get what was reserved
	resRaw, ok := data["fulfillment.reserve_inventory"]
	if !ok {
		return nil, nil // Nothing to release
	}
	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, nil
	}
	itemsRaw, ok := resMap["items"].([]any)
	if !ok {
		return nil, nil
	}

	for _, itemRaw := range itemsRaw {
		itemMap, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}
		productID, _ := itemMap["product_id"].(string)
		
		var quantity int
		if q, ok := itemMap["quantity"].(int); ok {
			quantity = q
		} else if qf, ok := itemMap["quantity"].(float64); ok {
			quantity = int(qf)
		}

		if productID == "" || quantity <= 0 {
			continue
		}

		inv, err := m.repo.GetInventoryByProductID(ctx, productID)
		if err == nil {
			inv.AvailableQuantity += quantity
			m.repo.SaveInventory(ctx, inv)
		}
	}

	slog.Warn("Saga Compensation: Inventory released")
	return nil, nil
}

func (m *Module) CreateShipment(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	oRaw, ok := data["order.create"]
	if !ok {
		return nil, fmt.Errorf("missing result from order.create step")
	}
	resMap, ok := oRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from order.create")
	}

	var orderID string
	if oGetter, ok := resMap["order"].(interface{ GetOrderID() string }); ok {
		orderID = oGetter.GetOrderID()
	} else if oMap, ok := resMap["order"].(map[string]any); ok {
		if id, ok := oMap["id"].(string); ok {
			orderID = id
		} else if id, ok := oMap["ID"].(string); ok {
			orderID = id
		}
	} else {
		// Fallback: try JSON marshal/unmarshal to extract ID
		dataBytes, _ := json.Marshal(resMap["order"])
		var temp struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(dataBytes, &temp); err == nil {
			orderID = temp.ID
		}
	}
	if orderID == "" {
		return nil, fmt.Errorf("missing order ID from order.create result")
	}

	s := &Shipment{
		ID:      "shp_" + uuid.New().String(),
		OrderID: orderID,
		Status:  ShipmentPending,
	}

	if err := m.repo.SaveShipment(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save shipment: %w", err)
	}

	slog.Info("Shipment created", "shipment_id", s.ID, "order_id", orderID)
	return map[string]any{"shipment": s}, nil
}

// ShipOrder simulates shipping the order (updates status to SHIPPED and adds tracking number).
func (m *Module) ShipOrder(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	
	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	shipmentID := getString(workflowInput, "shipment_id")
	trackingNumber := getString(workflowInput, "tracking_number")
	carrier := getString(workflowInput, "carrier")

	s, err := m.repo.GetShipment(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("shipment not found: %w", err)
	}

	s.Status = ShipmentShipped
	if trackingNumber != "" {
		s.TrackingNumber = trackingNumber
	}
	if carrier != "" {
		s.Carrier = carrier
	}

	if err := m.repo.SaveShipment(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to update shipment: %w", err)
	}

	slog.Info("Shipment shipped", "shipment_id", s.ID, "tracking_number", s.TrackingNumber)
	return map[string]any{"shipment": s}, nil
}

// ReserveInventoryStep wraps ReserveInventory to mdk.StepHandler.
func (m *Module) ReserveInventoryStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.ReserveInventory(sCtx.Ctx, map[string]any{
		"input": sCtx.Input,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// ReleaseInventoryStep wraps ReleaseInventory to mdk.StepHandler.
func (m *Module) ReleaseInventoryStep(sCtx mdk.StepContext) mdk.StepResult {
	_, err := m.ReleaseInventory(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{}
}

// CreateShipmentStep wraps CreateShipment to mdk.StepHandler.
func (m *Module) CreateShipmentStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.CreateShipment(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// ShipOrderStep wraps ShipOrder to mdk.StepHandler.
func (m *Module) ShipOrderStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.ShipOrder(sCtx.Ctx, map[string]any{
		"input": sCtx.Input,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
