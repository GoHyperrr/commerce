package product

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for the Product.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.product"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "product.create",
		Name: "Product Create",
		Steps: []mdk.Step{
			{ID: "validate", Name: "Validate Product", Uses: "product.validate_product"},
			{ID: "persist", Name: "Persist Product", Uses: "product.persist_product", DependsOn: []string{"validate"}},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "product.update",
		Name: "Product Update",
		Steps: []mdk.Step{
			{ID: "update", Name: "Update Details", Uses: "product.update_details"},
		},
	})

	// Register workflow step handlers
	_ = rt.Workflows().RegisterHandler("product.validate_product", m.ValidateProductStep)
	_ = rt.Workflows().RegisterHandler("product.persist_product", m.PersistProductStep)
	_ = rt.Workflows().RegisterHandler("product.update_details", m.UpdateProductDetailsStep)

	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Models() []any {
	return []any{&Product{}}
}

func (m *Module) Routes() []mdk.Route {
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
