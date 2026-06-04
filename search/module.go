package search

import (
	"context"

	"github.com/GoHyperrr/commerce/product"
	"github.com/GoHyperrr/mdk"
	"gorm.io/gorm"
)

// Module implements the mdk.Module interface for Search.
type Module struct {
	db      *gorm.DB
	prodMod *product.Module
	rt      mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.search"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.db = rt.DB()

	// Resolve product module dependency dynamically from the runtime
	if prodModVal, ok := rt.Module("commerce.product"); ok {
		if pm, ok := prodModVal.(*product.Module); ok {
			m.prodMod = pm
		}
	}

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "search.products",
		Name: "Search Products",
		Steps: []mdk.Step{
			{ID: "search_step", Name: "Search Catalog", Uses: "search.product_catalog"},
		},
	})

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("search.product_catalog", m.SearchProductsStep)

	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Models() []any {
	return []any{&SearchHistory{}}
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) SetProductModule(pm *product.Module) {
	m.prodMod = pm
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
