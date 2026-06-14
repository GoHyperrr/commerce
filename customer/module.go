package customer

import (
	"context"
	"encoding/json"
	"fmt"

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

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "customer.create",
		Name:        "customer.create",
		Description: "Create a new customer profile (supports both guest and registered profiles).",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":    map[string]any{"type": "string"},
				"email":   map[string]any{"type": "string"},
				"phone":   map[string]any{"type": "string"},
				"isGuest": map[string]any{"type": "boolean"},
				"userId":  map[string]any{"type": "string"},
			},
			"required": []string{"name", "email"},
		},
		Steps: []mdk.Step{
			{ID: "create", Name: "Create Profile", Uses: "customer.create_profile"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "customer.add_address",
		Name:        "customer.add_address",
		Description: "Add a shipping or billing address to a customer profile.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"customerId":   map[string]any{"type": "string"},
				"receiverName": map[string]any{"type": "string"},
				"phone":        map[string]any{"type": "string"},
				"line1":        map[string]any{"type": "string"},
				"line2":        map[string]any{"type": "string"},
				"city":         map[string]any{"type": "string"},
				"state":        map[string]any{"type": "string"},
				"zip":          map[string]any{"type": "string"},
				"country":      map[string]any{"type": "string"},
			},
			"required": []string{"customerId", "line1", "city", "state", "zip", "country"},
		},
		Steps: []mdk.Step{
			{ID: "add_address", Name: "Add Address", Uses: "customer.add_address_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "customer.get",
		Name:        "customer.get",
		Description: "Retrieve a customer profile by ID or email address.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":    map[string]any{"type": "string"},
				"email": map[string]any{"type": "string"},
			},
		},
		Steps: []mdk.Step{
			{ID: "get", Name: "Get Customer", Uses: "customer.get_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "customer.list",
		Name:        "customer.list",
		Description: "Retrieve a list of all customer profiles.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
		},
		Steps: []mdk.Step{
			{ID: "list", Name: "List Customers", Uses: "customer.list_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "customer.delete",
		Name:        "customer.delete",
		Description: "Delete a customer record by ID.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
			"required": []string{"id"},
		},
		Steps: []mdk.Step{
			{ID: "delete", Name: "Delete Customer", Uses: "customer.delete_step"},
		},
	})

	_ = rt.Workflows().RegisterHandler("customer.create_profile", m.CreateProfileStep)
	_ = rt.Workflows().RegisterHandler("customer.add_address_step", m.AddAddressStepHandler)
	_ = rt.Workflows().RegisterHandler("customer.get_step", m.GetCustomerStep)
	_ = rt.Workflows().RegisterHandler("customer.list_step", m.ListCustomersStep)
	_ = rt.Workflows().RegisterHandler("customer.delete_step", m.DeleteCustomerStepHandler)

	return nil
}

func (m *Module) GetCustomerStep(sCtx mdk.StepContext) mdk.StepResult {
	id, _ := sCtx.Input["id"].(string)
	email, _ := sCtx.Input["email"].(string)

	if id != "" {
		c, err := m.GetCustomer(sCtx.Ctx, id)
		if err != nil {
			return mdk.StepResult{Err: err}
		}
		return mdk.StepResult{Output: map[string]any{"customer": c}}
	} else if email != "" {
		var c Customer
		err := m.rt.DB().WithContext(sCtx.Ctx).First(&c, "email = ?", email).Error
		if err != nil {
			return mdk.StepResult{Err: err}
		}
		return mdk.StepResult{Output: map[string]any{"customer": &c}}
	}
	return mdk.StepResult{Err: fmt.Errorf("id or email is required")}
}

func (m *Module) ListCustomersStep(sCtx mdk.StepContext) mdk.StepResult {
	customers, err := m.ListCustomers(sCtx.Ctx)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"customers": customers}}
}

func (m *Module) DeleteCustomerStepHandler(sCtx mdk.StepContext) mdk.StepResult {
	id, _ := sCtx.Input["id"].(string)
	if id == "" {
		return mdk.StepResult{Err: fmt.Errorf("id required")}
	}
	ok, err := m.DeleteCustomer(sCtx.Ctx, id)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"success": ok}}
}

func (m *Module) CreateProfileStep(sCtx mdk.StepContext) mdk.StepResult {
	ba, err := json.Marshal(sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	var in CreateCustomerInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return mdk.StepResult{Err: err}
	}

	cust, err := m.CreateCustomer(sCtx.Ctx, in)
	if err != nil {
		return mdk.StepResult{Err: err}
	}

	return mdk.StepResult{Output: map[string]any{"customer": cust}}
}

func (m *Module) AddAddressStepHandler(sCtx mdk.StepContext) mdk.StepResult {
	ba, err := json.Marshal(sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	var in CreateAddressInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return mdk.StepResult{Err: err}
	}

	customerID, _ := sCtx.Input["customerId"].(string)
	if customerID == "" {
		return mdk.StepResult{Err: fmt.Errorf("customerId required")}
	}

	addr, err := m.AddCustomerAddress(sCtx.Ctx, customerID, in)
	if err != nil {
		return mdk.StepResult{Err: err}
	}

	return mdk.StepResult{Output: map[string]any{"address": addr}}
}

func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	customers, err := m.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var res []mdk.MCPResource
	for _, c := range customers {
		res = append(res, mdk.MCPResource{
			URI:         "customer://" + c.ID,
			Name:        "Customer: " + c.Name,
			Description: "Profile details for customer " + c.ID + " (" + c.Email + ")",
			MimeType:    "application/json",
		})
	}
	return res, nil
}

func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	var customerID string
	if n, err := fmt.Sscanf(uri, "customer://%s", &customerID); err != nil || n != 1 {
		return "", fmt.Errorf("invalid customer URI")
	}
	c, err := m.repo.GetByID(ctx, customerID)
	if err != nil {
		return "", err
	}
	dataBytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(dataBytes), nil
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
