package customer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

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

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) UpdateCustomer(ctx context.Context, id string, input UpdateCustomerInput) (*Customer, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
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
	results, err := executor.ExecuteSync(ctx, execID, "customer.update", workflowInput)
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

	return &domainRes, nil
}

func (m *Module) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	return m.repo.GetByID(ctx, id)
}

func (m *Module) ListCustomers(ctx context.Context) ([]*Customer, error) {
	return m.repo.List(ctx)
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
