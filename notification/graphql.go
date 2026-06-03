package notification

import (
	"context"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"listNotifications": m.ListNotifications,
	}
}

func (m *Module) Mutations() map[string]any {
	return nil
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) ListNotifications(ctx context.Context, recipient *string) ([]*model.Notification, error) {
	recip := ""
	if recipient != nil {
		recip = *recipient
	}

	notifs, err := m.repo.List(ctx, recip)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Notification, 0, len(notifs))
	for _, n := range notifs {
		res = append(res, mapNotificationToModel(n))
	}

	return res, nil
}

func mapNotificationToModel(n *Notification) *model.Notification {
	return &model.Notification{
		ID:        n.ID,
		Recipient: n.Recipient,
		Channel:   string(n.Channel),
		Subject:   n.Subject,
		Body:      n.Body,
		Status:    string(n.Status),
		CreatedAt: n.CreatedAt,
	}
}
