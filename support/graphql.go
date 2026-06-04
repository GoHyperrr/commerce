package support

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) CreateTicketResolver(ctx context.Context, customerID string, subject string, message string) (*Ticket, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	workflowInput := map[string]any{
		"customer_id": customerID,
		"subject":     subject,
		"message":     message,
	}

	execID := "tkt_wf_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "support.create", workflowInput)
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

	return t, nil
}

func (m *Module) AddTicketMessage(ctx context.Context, ticketID string, sender string, content string) (*Message, error) {
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

	return msg, nil
}

func (m *Module) GetTicket(ctx context.Context, id string) (*Ticket, error) {
	return m.repo.GetTicketByID(ctx, id)
}

func (m *Module) ListCustomerTickets(ctx context.Context, customerID string) ([]*Ticket, error) {
	return m.repo.ListTicketsByCustomerID(ctx, customerID)
}
