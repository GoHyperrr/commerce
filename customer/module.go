package customer

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Customer.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

// NewModule creates a new instance of the Customer module.
func NewModule() *Module {
	return &Module{}
}

// ID returns the unique identifier for the customer module.
func (m *Module) ID() string {
	return "commerce.customer"
}

// Init initializes the customer module, subscribing to identity events to seed customer profiles.
func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Subscribe to user creation to seed a business profile automatically.
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
			ID:      "cust_" + userID,
			UserID:  actorID,
			IsGuest: false,
			Name:    name,
			Email:   email,
		}

		if err := m.repo.Save(ctx, c); err != nil {
			rt.Logger().Error("failed to create customer from user_created event", "error", err)
			return err
		}

		rt.Logger().Info("Customer profile seeded for registered user", "id", c.ID)
		return nil
	})

	return nil
}

// Models registers GORM database structures for database migrations.
func (m *Module) Models() []any {
	return []any{&Customer{}, &Address{}}
}

// Routes returns HTTP endpoints (none for this module).
func (m *Module) Routes() []mdk.Route {
	return nil
}

// Shutdown cleans up resources.
func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

// Repo returns the customer Repository.
func (m *Module) Repo() *Repository {
	return m.repo
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
