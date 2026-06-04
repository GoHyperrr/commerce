package fulfillment

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Fulfillment.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.fulfillment"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "fulfillment.ship_order",
		Name: "Fulfillment Ship Order",
		Steps: []mdk.Step{
			{
				ID:   "ship",
				Name: "Ship Order",
				Uses: "fulfillment.ship_order",
			},
		},
	})

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("fulfillment.reserve_inventory", m.ReserveInventoryStep)
	_ = rt.Workflows().RegisterHandler("fulfillment.release_inventory", m.ReleaseInventoryStep)
	_ = rt.Workflows().RegisterHandler("fulfillment.create_shipment", m.CreateShipmentStep)
	_ = rt.Workflows().RegisterHandler("fulfillment.ship_order", m.ShipOrderStep)

	return nil
}

func (m *Module) Models() []any {
	return []any{&Inventory{}, &Shipment{}}
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
