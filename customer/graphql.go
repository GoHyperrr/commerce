package customer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/GoHyperrr/hyperrr/pkg/workflow"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getCustomer":   m.GetCustomer,
		"listCustomers": m.ListCustomers,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"updateCustomer": m.UpdateCustomer,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) UpdateCustomer(ctx context.Context, id string, input model.UpdateCustomerInput) (*model.Customer, error) {
	wf := &workflow.Workflow{
		Name: "customer.update",
		Steps: []workflow.Step{
			{ID: "customer.update_details", Uses: "customer.update_details"},
		},
	}

	workflowInput := map[string]any{
		"id": id,
	}
	if input.Name != nil {
		workflowInput["name"] = *input.Name
	}
	if input.Email != nil {
		workflowInput["email"] = *input.Email
	}

	execID := "update_cust_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["customer.update_details"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve updated customer from workflow results")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from update step")
	}

	var domainRes Customer
	if err := decodeResult(resMap["customer"], &domainRes); err != nil {
		return nil, fmt.Errorf("invalid customer type in results: %w", err)
	}

	return mapCustomerToModel(&domainRes), nil
}

func (m *Module) GetCustomer(ctx context.Context, id string) (*model.Customer, error) {
	c, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return mapCustomerToModel(c), nil
}

func (m *Module) ListCustomers(ctx context.Context) ([]*model.Customer, error) {
	list, err := m.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var res []*model.Customer
	for _, c := range list {
		res = append(res, mapCustomerToModel(c))
	}

	return res, nil
}

func mapCustomerToModel(c *Customer) *model.Customer {
	res := &model.Customer{
		ID:      c.ID,
		UserID:  c.UserID,
		Name:    c.Name,
		Email:   c.Email,
		Persona: &c.Persona,
	}

	for _, a := range c.Addresses {
		res.Addresses = append(res.Addresses, &model.Address{
			ID:      a.ID,
			Line1:   a.Line1,
			Line2:   &a.Line2,
			City:    a.City,
			State:   a.State,
			Zip:     a.Zip,
			Country: a.Country,
		})
	}

	return res
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
