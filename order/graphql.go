package order

import (
	"context"
	"encoding/json"
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

func (m *Module) CreateOrderFromCart(ctx context.Context, cartID string) (*model.Order, error) {
	cartModRaw, ok := registry.Get("commerce.cart")
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

	workflowInput := map[string]any{
		"customer_id": c.CustomerID,
		"cart_id":     c.ID,
		"items":       items,
	}

	wf, err := m.deps.Registry.Get("fulfillment.v1")
	if err != nil {
		return nil, err
	}

	execID := "fulfill_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
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

	return mapOrderToModel(&o), nil
}

func (m *Module) GetOrder(ctx context.Context, id string) (*model.Order, error) {
	o, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapOrderToModel(o), nil
}

func (m *Module) ListOrders(ctx context.Context) ([]*model.Order, error) {
	orders, err := m.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		res = append(res, mapOrderToModel(o))
	}
	return res, nil
}

func (m *Module) ListCustomerOrders(ctx context.Context, customerID string) ([]*model.Order, error) {
	orders, err := m.repo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		res = append(res, mapOrderToModel(o))
	}
	return res, nil
}

func mapOrderToModel(o *Order) *model.Order {
	res := &model.Order{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		Status:     string(o.Status),
		TotalPrice: o.TotalPrice,
	}
	for _, item := range o.Items {
		res.Items = append(res.Items, &model.OrderItem{
			ID:        item.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		})
	}
	return res
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
