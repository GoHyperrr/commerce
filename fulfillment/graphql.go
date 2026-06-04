package fulfillment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getInventory":       m.GetInventory,
		"getShipment":        m.GetShipment,
		"getShipmentByOrder": m.GetShipmentByOrder,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"updateShipmentStatus": m.UpdateShipmentStatus,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) UpdateShipmentStatus(ctx context.Context, shipmentID string, trackingNumber *string, carrier *string) (*Shipment, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	workflowInput := map[string]any{
		"shipment_id": shipmentID,
	}
	if trackingNumber != nil {
		workflowInput["tracking_number"] = *trackingNumber
	}
	if carrier != nil {
		workflowInput["carrier"] = *carrier
	}

	execID := "ship_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "fulfillment.ship_order", workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["ship"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve updated shipment from workflow results")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from ship step")
	}

	var domainRes Shipment
	if err := decodeResult(resMap["shipment"], &domainRes); err != nil {
		return nil, fmt.Errorf("invalid shipment type in results: %w", err)
	}

	return &domainRes, nil
}

func (m *Module) GetInventory(ctx context.Context, productID string) (*Inventory, error) {
	return m.repo.GetInventoryByProductID(ctx, productID)
}

func (m *Module) GetShipment(ctx context.Context, id string) (*Shipment, error) {
	return m.repo.GetShipment(ctx, id)
}

func (m *Module) GetShipmentByOrder(ctx context.Context, orderID string) (*Shipment, error) {
	return m.repo.GetShipmentByOrderID(ctx, orderID)
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
