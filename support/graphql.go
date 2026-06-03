package support

import (
	"context"
	"fmt"
	"time"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getTicket":           m.GetTicket,
		"listCustomerTickets": m.ListCustomerTickets,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"createTicket":     m.CreateTicketResolver,
		"addTicketMessage": m.AddTicketMessage,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) CreateTicketResolver(ctx context.Context, customerID string, subject string, message string) (*model.Ticket, error) {
	wf, err := m.deps.Registry.Get("support.create")
	if err != nil {
		return nil, err
	}

	workflowInput := map[string]any{
		"customer_id": customerID,
		"subject":     subject,
		"message":     message,
	}

	execID := "tkt_wf_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["ticket"]
	if !ok {
		return nil, fmt.Errorf("missing ticket from workflow results")
	}
	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from ticket step")
	}
	t, ok := resMap["ticket"].(*Ticket)
	if !ok {
		return nil, fmt.Errorf("failed to retrieve ticket from results")
	}

	return mapTicketToModel(t), nil
}

func (m *Module) AddTicketMessage(ctx context.Context, ticketID string, sender string, content string) (*model.Message, error) {
	msg := &Message{
		ID:        "msg_" + uuid.New().String(),
		TicketID:  ticketID,
		Sender:    SenderType(sender),
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := m.repo.SaveMessage(ctx, msg); err != nil {
		return nil, err
	}

	return mapMessageToModel(msg), nil
}

func (m *Module) GetTicket(ctx context.Context, id string) (*model.Ticket, error) {
	t, err := m.repo.GetTicketByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapTicketToModel(t), nil
}

func (m *Module) ListCustomerTickets(ctx context.Context, customerID string) ([]*model.Ticket, error) {
	tickets, err := m.repo.ListTicketsByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Ticket, 0, len(tickets))
	for _, t := range tickets {
		res = append(res, mapTicketToModel(t))
	}
	return res, nil
}

func mapTicketToModel(t *Ticket) *model.Ticket {
	res := &model.Ticket{
		ID:         t.ID,
		CustomerID: t.CustomerID,
		Subject:    t.Subject,
		Status:     string(t.Status),
		CreatedAt:  t.CreatedAt,
	}
	for _, m := range t.Messages {
		res.Messages = append(res.Messages, mapMessageToModel(&m))
	}
	return res
}

func mapMessageToModel(m *Message) *model.Message {
	return &model.Message{
		ID:        m.ID,
		TicketID:  m.TicketID,
		Sender:    string(m.Sender),
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}
}
