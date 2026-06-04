package cart

import (
	"context"

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
		ID:   "cart.add",
		Name: "Cart Add Item",
		Steps: []mdk.Step{
			{
				ID:      "add",
				Name:    "Add Item",
				Handler: m.AddItemStep,
			},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "cart.remove",
		Name: "Cart Remove Item",
		Steps: []mdk.Step{
			{
				ID:      "remove",
				Name:    "Remove Item",
				Handler: m.RemoveItemStep,
			},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "cart.checkout",
		Name: "Cart Checkout",
		Steps: []mdk.Step{
			{
				ID:      "checkout",
				Name:    "Checkout",
				Handler: m.CheckoutStep,
			},
		},
	})

	_ = rt.Workflows().RegisterHandler("cart.add_item", m.AddItemStep)
	_ = rt.Workflows().RegisterHandler("cart.remove_item", m.RemoveItemStep)

	return nil
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

