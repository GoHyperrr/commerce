package customer

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Customer.
type Module struct {
	repo      *Repository
	brain     *MLBrainV2
	projector mdk.Projector
	rt        mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.customer"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Try to resolve Projector from registry if not explicitly set
	if m.projector == nil {
		if ctxModVal, ok := rt.Module("core.context"); ok {
			if provider, ok := ctxModVal.(mdk.ProjectorProvider); ok {
				m.projector = provider.Projector()
				m.brain = NewMLBrainV2(m.projector)
			}
		}
	}

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "customer.segmentation",
		Name: "Customer Segmentation",
		Steps: []mdk.Step{
			{ID: "calculate", Name: "Calculate Persona", Uses: "customer.calculate_persona"},
			{ID: "update", Name: "Update Persona", Uses: "customer.update_persona", DependsOn: []string{"calculate"}},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "customer.update",
		Name: "Customer Update",
		Steps: []mdk.Step{
			{ID: "customer.update_details", Name: "Update Customer Details", Uses: "customer.update_details"},
		},
	})

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("customer.calculate_persona", m.CalculatePersonaStep)
	_ = rt.Workflows().RegisterHandler("customer.update_persona", m.UpdatePersonaStep)
	_ = rt.Workflows().RegisterHandler("customer.update_details", m.UpdateCustomerDetailsStep)

	// Subscribe to user creation to create a business profile
	_, _ = rt.Bus().Subscribe("identity", "user_created", func(ctx context.Context, event mdk.Event) error {
		actorID := getString(event.Payload, "actor_id")
		userID := getString(event.Payload, "user_id")
		if userID == "" {
			userID = actorID
		}
		name := getString(event.Payload, "name")
		email := getString(event.Payload, "email")

		if actorID == "" {
			return nil
		}

		c := &Customer{
			ID:     "cust_" + userID,
			UserID: actorID,
			Name:   name,
			Email:  email,
		}

		if err := m.repo.Save(ctx, c); err != nil {
			rt.Logger().Error("failed to create customer from event", "error", err)
			return err
		}

		rt.Logger().Info("Customer profile created for user", "id", c.ID)
		return nil
	})

	// Subscribe to order completions to trigger ML segmentation
	_, _ = rt.Bus().Subscribe("order", "completed", func(ctx context.Context, event mdk.Event) error {
		customerID := getString(event.Payload, "customer_id")
		if customerID == "" {
			return nil
		}

		workflowID := "seg_" + customerID
		_ = workflowID
		go func() {
			if _, err := rt.Workflows().Execute(ctx, "customer.segmentation", event.Payload); err != nil {
				rt.Logger().Error("background segmentation failed", "customer_id", customerID, "error", err)
			}
		}()
		return nil 
	})

	return nil
}

func (m *Module) Models() []any {
	return []any{&Customer{}, &Address{}}
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

func (m *Module) SetProjector(p mdk.Projector) {
	m.projector = p
	m.brain = NewMLBrainV2(p)
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
