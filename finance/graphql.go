package finance

import (
	"context"
)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getPayment":        m.GetPayment,
		"listOrderPayments": m.ListOrderPayments,
	}
}

func (m *Module) Mutations() map[string]any {
	return nil
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) GetPayment(ctx context.Context, id string) (*Payment, error) {
	return m.repo.GetByID(ctx, id)
}

func (m *Module) ListOrderPayments(ctx context.Context, orderID string) ([]*Payment, error) {
	return m.repo.ListByOrderID(ctx, orderID)
}
