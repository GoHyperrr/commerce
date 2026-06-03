package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/GoHyperrr/commerce/order"
	"github.com/GoHyperrr/hyperrr/api/graph/model"
	ctxEngine "github.com/GoHyperrr/hyperrr/pkg/ctxengine"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/GoHyperrr/hyperrr/pkg/workflow"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getSystemStats": m.GetSystemStats,
		"getSalesStats":  m.GetSalesStats,
	}
}

func (m *Module) Mutations() map[string]any {
	return nil
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) GetSystemStats(ctx context.Context) (*model.SystemStats, error) {
	mod, ok := registry.Get("core.context")
	if !ok {
		return nil, fmt.Errorf("core.context module not found")
	}
	ctxMod, ok := mod.(*ctxEngine.Module)
	if !ok {
		return nil, fmt.Errorf("invalid type for core.context module")
	}
	projector := ctxMod.Projector()
	if projector == nil {
		return nil, fmt.Errorf("analytics module not initialized with projector")
	}

	lineages := projector.ListLineages()

	total := len(lineages)
	if total == 0 {
		return &model.SystemStats{}, nil
	}

	success := 0
	failed := 0
	var totalDuration time.Duration

	for _, l := range lineages {
		if l.GetState() == workflow.StateCompleted {
			success++
		} else if l.GetState() == workflow.StateFailed {
			failed++
		}
		if l.GetEndedAt() != nil && !l.GetEndedAt().IsZero() {
			totalDuration += l.GetEndedAt().Sub(l.GetStartedAt())
		}
	}

	return &model.SystemStats{
		TotalWorkflows:     total,
		SuccessRate:        (float64(success) / float64(total)) * 100,
		FailureRate:        (float64(failed) / float64(total)) * 100,
		AvgExecutionTimeMs: float64(totalDuration.Milliseconds()) / float64(total),
	}, nil
}

func (m *Module) GetSalesStats(ctx context.Context) (*model.SalesStats, error) {
	mod, ok := registry.Get("core.context")
	if !ok {
		return nil, fmt.Errorf("core.context module not found")
	}
	ctxMod, ok := mod.(*ctxEngine.Module)
	if !ok {
		return nil, fmt.Errorf("invalid type for core.context module")
	}
	projector := ctxMod.Projector()
	if projector == nil {
		return nil, fmt.Errorf("analytics module not initialized with projector")
	}

	lineages := projector.ListLineages()

	var totalRevenue float64
	orderCount := 0
	orderIDs := make(map[string]bool)

	for _, l := range lineages {
		// Look for successful fulfillment workflows
		if l.GetName() == "fulfillment.v1" && l.GetState() == workflow.StateCompleted {
			// Extract revenue from lineage events
			conc, ok := l.(*ctxEngine.Lineage)
			if ok {
				for _, ev := range conc.Events {
					if ev.Type == "order.paid" {
						if p, ok := ev.Payload.(map[string]any); ok {
							if total, ok := p["total_price"].(float64); ok {
								totalRevenue += total
								orderCount++
								if id, ok := p["order_id"].(string); ok {
									orderIDs[id] = true
								}
								break
							}
						}
					}
				}
			}
		}
	}

	// Secondary check: Database reconciliation
	ordModRaw, ok := registry.Get("commerce.order")
	if ok {
		if ordMod, ok := ordModRaw.(*order.Module); ok {
			orders, err := ordMod.Repo().List(ctx)
			if err == nil {
				for _, o := range orders {
					if (o.Status == order.OrderPaid || o.Status == order.OrderFulfilled) && !orderIDs[o.ID] {
						totalRevenue += o.TotalPrice
						orderCount++
						orderIDs[o.ID] = true
					}
				}
			}
		}
	}

	avg := 0.0
	if orderCount > 0 {
		avg = totalRevenue / float64(orderCount)
	}

	return &model.SalesStats{
		TotalRevenue:  totalRevenue,
		OrderCount:    orderCount,
		AvgOrderValue: avg,
	}, nil
}
