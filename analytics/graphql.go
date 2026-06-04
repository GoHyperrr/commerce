package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/GoHyperrr/commerce/order"
	"github.com/GoHyperrr/mdk"
)

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

func (m *Module) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	mod, ok := m.rt.Module("core.context")
	if !ok {
		return nil, fmt.Errorf("core.context module not found")
	}
	prov, ok := mod.(mdk.ProjectorProvider)
	if !ok {
		return nil, fmt.Errorf("invalid type for core.context module")
	}
	projector := prov.Projector()
	if projector == nil {
		return nil, fmt.Errorf("analytics module not initialized with projector")
	}

	lineages := projector.ListLineages()

	total := len(lineages)
	if total == 0 {
		return &SystemStats{}, nil
	}

	success := 0
	failed := 0
	var totalDuration time.Duration

	for _, l := range lineages {
		if l.GetState() == "COMPLETED" {
			success++
		} else if l.GetState() == "FAILED" {
			failed++
		}
		if l.GetEndedAt() != nil && !l.GetEndedAt().IsZero() {
			totalDuration += l.GetEndedAt().Sub(l.GetStartedAt())
		}
	}

	return &SystemStats{
		TotalWorkflows:     total,
		SuccessRate:        (float64(success) / float64(total)) * 100,
		FailureRate:        (float64(failed) / float64(total)) * 100,
		AvgExecutionTimeMs: float64(totalDuration.Milliseconds()) / float64(total),
	}, nil
}

func (m *Module) GetSalesStats(ctx context.Context) (*SalesStats, error) {
	mod, ok := m.rt.Module("core.context")
	if !ok {
		return nil, fmt.Errorf("core.context module not found")
	}
	prov, ok := mod.(mdk.ProjectorProvider)
	if !ok {
		return nil, fmt.Errorf("invalid type for core.context module")
	}
	projector := prov.Projector()
	if projector == nil {
		return nil, fmt.Errorf("analytics module not initialized with projector")
	}

	lineages := projector.ListLineages()

	var totalRevenue float64
	orderCount := 0
	orderIDs := make(map[string]bool)

	for _, l := range lineages {
		// Look for successful fulfillment workflows
		if l.GetName() == "fulfillment.v1" && l.GetState() == "COMPLETED" {
			// Extract revenue from lineage events
			for _, ev := range l.GetEvents() {
				if ev.Type == "order.paid" {
					if total, ok := ev.Payload["total_price"].(float64); ok {
						totalRevenue += total
						orderCount++
						if id, ok := ev.Payload["order_id"].(string); ok {
							orderIDs[id] = true
						}
						break
					}
				}
			}
		}
	}

	// Secondary check: Database reconciliation
	ordModRaw, ok := m.rt.Module("commerce.order")
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

	return &SalesStats{
		TotalRevenue:  totalRevenue,
		OrderCount:    orderCount,
		AvgOrderValue: avg,
	}, nil
}
