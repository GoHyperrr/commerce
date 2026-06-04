package order

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Order.
type Module struct {
	repo *Repository
	bus  mdk.EventBus
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.order"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.repo = NewRepository(rt.DB())
	m.bus = rt.Bus()
	m.rt = rt

	// Register Fulfillment Saga
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "fulfillment.v1",
		Name:        "Fulfillment Workflow",
		Description: "Orchestrates the full order lifecycle including inventory reservation, payment processing, and shipment creation.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"customer_id": map[string]any{"type": "string"},
				"items": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"product_id": map[string]any{"type": "string"},
							"quantity":   map[string]any{"type": "integer"},
						},
					},
				},
			},
			"required": []string{"customer_id", "items"},
		},
		Steps: []mdk.Step{
			{
				ID:   "fulfillment.reserve_inventory",
				Uses: "fulfillment.reserve_inventory",
				Saga: &mdk.Saga{Uses: "fulfillment.release_inventory"},
			},
			{
				ID:        TaskCreateOrder,
				Uses:      TaskCreateOrder,
				Saga:      &mdk.Saga{Uses: TaskCompensatePayment},
				DependsOn: []string{"fulfillment.reserve_inventory"},
			},
			{
				ID:        "finance.process_payment",
				Uses:      "finance.process_payment",
				DependsOn: []string{TaskCreateOrder},
				Saga:      &mdk.Saga{Uses: "finance.compensate_payment"},
			},
			{
				ID:        "fulfillment.create_shipment",
				Uses:      "fulfillment.create_shipment",
				DependsOn: []string{"finance.process_payment"},
			},
			{
				ID:        TaskFinalizeOrder,
				Uses:      TaskFinalizeOrder,
				DependsOn: []string{"fulfillment.create_shipment"},
			},
			{
				ID:        "marketing.add_loyalty_points",
				Uses:      "marketing.add_loyalty_points",
				DependsOn: []string{TaskFinalizeOrder},
			},
		},
	})

	// Register Handlers
	_ = rt.Workflows().RegisterHandler(TaskCreateOrder, m.CreateOrderStep)
	_ = rt.Workflows().RegisterHandler(TaskFinalizeOrder, m.FinalizeOrderStep)
	_ = rt.Workflows().RegisterHandler(TaskCompensatePayment, m.CompensatePaymentStep)

	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Models() []any {
	return []any{&Order{}, &OrderItem{}}
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) Repo() *Repository {
	return m.repo
}

func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	if m.repo == nil {
		return nil, nil
	}
	orders, err := m.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var res []mdk.MCPResource
	for _, o := range orders {
		res = append(res, mdk.MCPResource{
			URI:         "order://" + o.ID + "/status",
			Name:        "Order Status: " + o.ID,
			Description: "Real-time fulfillment status of order " + o.ID,
			MimeType:    "application/json",
		})
	}
	return res, nil
}

func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	if m.repo == nil {
		return "", fmt.Errorf("order repository not initialized")
	}
	var orderID string
	n, err := fmt.Sscanf(uri, "order://%s", &orderID)
	if err != nil || n != 1 {
		return "", fmt.Errorf("invalid URI format")
	}
	orderID = strings.TrimSuffix(orderID, "/status")

	o, err := m.repo.GetByID(ctx, orderID)
	if err != nil {
		return "", err
	}

	data := map[string]any{
		"order_id":    o.ID,
		"customer_id": o.CustomerID,
		"status":      string(o.Status),
		"total_price": o.TotalPrice,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
