package cart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

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

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) AddItemToCart(ctx context.Context, cartID string, input AddItemInput) (*Cart, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}
	workflowInput := map[string]any{
		"cart_id":    cartID,
		"product_id": input.ProductID,
		"quantity":   input.Quantity,
		"price":      input.Price,
	}

	execID := "add_item_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "cart.add", workflowInput)
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

	return &domainRes, nil
}

func (m *Module) RemoveItemFromCart(ctx context.Context, cartID string, itemID string) (*Cart, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	workflowInput := map[string]any{
		"cart_id": cartID,
		"item_id": itemID,
	}

	execID := "remove_item_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "cart.remove", workflowInput)
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

	return &domainRes, nil
}

func (m *Module) CheckoutCart(ctx context.Context, cartID string) (bool, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return false, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	workflowInput := map[string]any{
		"cart_id": cartID,
	}

	execID := "checkout_" + uuid.New().String()
	_, err := executor.ExecuteSync(ctx, execID, "cart.checkout", workflowInput)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *Module) GetCart(ctx context.Context, id string) (*Cart, error) {
	return m.repo.GetByID(ctx, id)
}

func (m *Module) GetActiveCart(ctx context.Context, customerID string) (*Cart, error) {
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
		return newCart, nil
	}
	return c, nil
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
