package cart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Cart.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.cart"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.repo = NewRepository(rt.DB())
	m.rt = rt

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "cart.add",
		Name:        "cart.add",
		Description: "Add a product item to a shopping cart.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cart_id":    map[string]any{"type": "string"},
				"product_id": map[string]any{"type": "string"},
				"quantity":   map[string]any{"type": "integer"},
				"price":      map[string]any{"type": "number"},
			},
			"required": []string{"cart_id", "product_id", "quantity"},
		},
		Steps: []mdk.Step{
			{
				ID:   "add",
				Name: "Add Item",
				Uses: "cart.add_item",
			},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "cart.remove",
		Name:        "cart.remove",
		Description: "Remove an item from a shopping cart.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cart_id": map[string]any{"type": "string"},
				"item_id": map[string]any{"type": "string"},
			},
			"required": []string{"cart_id", "item_id"},
		},
		Steps: []mdk.Step{
			{
				ID:   "remove",
				Name: "Remove Item",
				Uses: "cart.remove_item",
			},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "cart.checkout",
		Name:        "cart.checkout",
		Description: "Finalize cart checkout and set shipping/billing addresses, payment method, and shipping carrier/method.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cart_id":             map[string]any{"type": "string"},
				"shipping_address_id": map[string]any{"type": "string"},
				"billing_address_id":  map[string]any{"type": "string"},
				"payment_method":      map[string]any{"type": "string"},
				"shipping_carrier":    map[string]any{"type": "string"},
				"shipping_method":     map[string]any{"type": "string"},
			},
			"required": []string{"cart_id"},
		},
		Steps: []mdk.Step{
			{
				ID:   "checkout",
				Name: "Checkout",
				Uses: "cart.checkout",
			},
		},
	})

	_ = rt.Workflows().RegisterHandler("cart.add_item", m.AddItemStep)
	_ = rt.Workflows().RegisterHandler("cart.remove_item", m.RemoveItemStep)
	_ = rt.Workflows().RegisterHandler("cart.checkout", m.CheckoutStep)

	return nil
}

func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	// Let's query active or active/completed carts
	var carts []*Cart
	err := m.rt.DB().WithContext(ctx).Order("updated_at desc").Limit(50).Find(&carts).Error
	if err != nil {
		return nil, err
	}
	var res []mdk.MCPResource
	for _, c := range carts {
		res = append(res, mdk.MCPResource{
			URI:         "cart://" + c.ID,
			Name:        "Cart: " + c.ID,
			Description: fmt.Sprintf("Shopping cart details for customer %s (Status: %s)", c.CustomerID, c.Status),
			MimeType:    "application/json",
		})
	}
	return res, nil
}

func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	var cartID string
	if n, err := fmt.Sscanf(uri, "cart://%s", &cartID); err != nil || n != 1 {
		return "", fmt.Errorf("invalid cart URI")
	}
	c, err := m.repo.GetByID(ctx, cartID)
	if err != nil {
		return "", err
	}
	dataBytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(dataBytes), nil
}

func (m *Module) Models() []any {
	return []any{&Cart{}, &CartItem{}}
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Repo() *Repository {
	return m.repo
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}

