package support

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Support.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.support"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "support.create",
		Name: "Support Create Ticket",
		Steps: []mdk.Step{
			{ID: "ticket", Name: "Create Ticket", Uses: "support.create_ticket"},
			{ID: "ai_response", Name: "Dispatch AI Response", Uses: "support.dispatch_ai_response", DependsOn: []string{"ticket"}},
		},
	})

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("support.create_ticket", m.CreateTicketStep)
	_ = rt.Workflows().RegisterHandler("support.dispatch_ai_response", m.DispatchAIResponseStep)

	return nil
}

func (m *Module) Models() []any {
	return []any{&Ticket{}, &Message{}}
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
