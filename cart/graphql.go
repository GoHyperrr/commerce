package cart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getCart":       m.GetCart,
		"getActiveCart": m.GetActiveCart,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"addItemToCart":      m.AddItemToCart,
		"removeItemFromCart": m.RemoveItemFromCart,
		"checkoutCart":       m.CheckoutCart,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) AddItemToCart(ctx context.Context, cartID string, input model.AddItemInput) (*model.Cart, error) {
	wf, err := m.deps.Registry.Get("cart.add")
	if err != nil {
		return nil, err
	}

	workflowInput := map[string]any{
		"cart_id":    cartID,
		"product_id": input.ProductID,
		"quantity":   input.Quantity,
		"price":      input.Price,
	}

	execID := "add_item_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["add"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve updated cart from workflow results")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from add step")
	}

	var domainRes Cart
	if err := decodeResult(resMap["cart"], &domainRes); err != nil {
		return nil, fmt.Errorf("invalid cart type in results: %w", err)
	}

	return mapCartToModel(&domainRes), nil
}

func (m *Module) RemoveItemFromCart(ctx context.Context, cartID string, itemID string) (*model.Cart, error) {
	wf, err := m.deps.Registry.Get("cart.remove")
	if err != nil {
		return nil, err
	}

	workflowInput := map[string]any{
		"cart_id": cartID,
		"item_id": itemID,
	}

	execID := "remove_item_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["remove"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve updated cart from workflow results")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from remove step")
	}

	var domainRes Cart
	if err := decodeResult(resMap["cart"], &domainRes); err != nil {
		return nil, fmt.Errorf("invalid cart type in results: %w", err)
	}

	return mapCartToModel(&domainRes), nil
}

func (m *Module) CheckoutCart(ctx context.Context, cartID string) (bool, error) {
	wf, err := m.deps.Registry.Get("cart.checkout")
	if err != nil {
		return false, err
	}

	workflowInput := map[string]any{
		"cart_id": cartID,
	}

	execID := "checkout_" + uuid.New().String()
	_, err = m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *Module) GetCart(ctx context.Context, id string) (*model.Cart, error) {
	c, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapCartToModel(c), nil
}

func (m *Module) GetActiveCart(ctx context.Context, customerID string) (*model.Cart, error) {
	c, err := m.repo.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		// If not found, create a new one
		newCart := &Cart{
			ID:         "cart_" + uuid.New().String(),
			CustomerID: customerID,
			Status:     CartActive,
		}
		if err := m.repo.Save(ctx, newCart); err != nil {
			return nil, err
		}
		return mapCartToModel(newCart), nil
	}
	return mapCartToModel(c), nil
}

func mapCartToModel(c *Cart) *model.Cart {
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
	return res
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
