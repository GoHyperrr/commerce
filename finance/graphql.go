package finance

import (
	"context"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

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

func (m *Module) GetPayment(ctx context.Context, id string) (*model.Payment, error) {
	p, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapPaymentToModel(p), nil
}

func (m *Module) ListOrderPayments(ctx context.Context, orderID string) ([]*model.Payment, error) {
	payments, err := m.repo.ListByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Payment, 0, len(payments))
	for _, p := range payments {
		res = append(res, mapPaymentToModel(p))
	}
	return res, nil
}

func mapPaymentToModel(p *Payment) *model.Payment {
	return &model.Payment{
		ID:      p.ID,
		OrderID: p.OrderID,
		Amount:  p.Amount,
		Status:  string(p.Status),
	}
}
