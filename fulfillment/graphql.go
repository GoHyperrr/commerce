package fulfillment

import (
	"context"
	"fmt"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

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

func (m *Module) UpdateShipmentStatus(ctx context.Context, shipmentID string, trackingNumber *string, carrier *string) (*model.Shipment, error) {
	wf, err := m.deps.Registry.Get("fulfillment.ship_order")
	if err != nil {
		return nil, err
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
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
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

	domainRes, ok := resMap["shipment"].(*Shipment)
	if !ok {
		return nil, fmt.Errorf("invalid shipment type in results")
	}

	return mapShipmentToModel(domainRes), nil
}

func (m *Module) GetInventory(ctx context.Context, productID string) (*model.Inventory, error) {
	inv, err := m.repo.GetInventoryByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	return mapInventoryToModel(inv), nil
}

func (m *Module) GetShipment(ctx context.Context, id string) (*model.Shipment, error) {
	s, err := m.repo.GetShipment(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapShipmentToModel(s), nil
}

func (m *Module) GetShipmentByOrder(ctx context.Context, orderID string) (*model.Shipment, error) {
	s, err := m.repo.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return mapShipmentToModel(s), nil
}

func mapInventoryToModel(inv *Inventory) *model.Inventory {
	return &model.Inventory{
		ID:                inv.ID,
		ProductID:         inv.ProductID,
		AvailableQuantity: inv.AvailableQuantity,
	}
}

func mapShipmentToModel(s *Shipment) *model.Shipment {
	var trackingNumber *string
	if s.TrackingNumber != "" {
		trackingNumber = &s.TrackingNumber
	}
	var carrier *string
	if s.Carrier != "" {
		carrier = &s.Carrier
	}
	return &model.Shipment{
		ID:             s.ID,
		OrderID:        s.OrderID,
		Status:         string(s.Status),
		TrackingNumber: trackingNumber,
		Carrier:        carrier,
	}
}
