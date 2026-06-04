package customer

import (
	"context"
	"log/slog"

	"github.com/GoHyperrr/mdk"
)

const (
	StateCompleted = "COMPLETED"
	StateFailed    = "FAILED"
)

// MLBrainV2 analyzes a customer's history via the Context Engine to assign a persona.
type MLBrainV2 struct {
	projector mdk.Projector
}

func NewMLBrainV2(p mdk.Projector) *MLBrainV2 {
	return &MLBrainV2{projector: p}
}

func (m *MLBrainV2) Analyze(ctx context.Context, customerID string) (string, error) {
	if m.projector == nil {
		return "NEWBIE", nil
	}

	// Query for successful orders
	orders := m.projector.QueryLineages(func(l mdk.LineageData) bool {
		return l.GetName() == "fulfillment.v1" && l.GetState() == StateCompleted
	})
	
	// Query for failures
	failures := m.projector.QueryLineages(func(l mdk.LineageData) bool {
		return l.GetState() == StateFailed
	})
	
	orderCount := len(orders)
	failureCount := len(failures)

	slog.Info("AI Brain analyzing customer", "customer_id", customerID, "orders", orderCount, "failures", failureCount)

	if orderCount > 5 {
		return "WHALE", nil
	}
	if orderCount > 2 {
		return "GOLD", nil
	}
	if failureCount > 2 {
		return "FRUSTRATED", nil
	}

	return "REGULAR", nil
}
