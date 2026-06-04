package customer

import (
	"context"
	"fmt"

	"github.com/GoHyperrr/mdk"
)

// CalculatePersona determines a customer's persona using the MLBrainV2.
func (m *Module) CalculatePersona(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	customerID := getString(workflowInput, "customer_id")
	if customerID == "" {
		return nil, fmt.Errorf("customer_id is required")
	}

	if m.brain == nil {
		return nil, fmt.Errorf("ML brain not initialized")
	}

	persona, err := m.brain.Analyze(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"customer_id": customerID,
		"persona":     persona,
	}, nil
}

// UpdatePersona saves the calculated persona to the customer record.
func (m *Module) UpdatePersona(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	personaData, ok := data["calculate"].(map[string]any)
	if !ok {
		// Fallback to older step name if needed, but the current DAG uses 'calculate'
		personaData, ok = data["customer.calculate_persona"].(map[string]any)
	}
	if !ok {
		return nil, fmt.Errorf("missing persona data")
	}

	customerID := getString(personaData, "customer_id")
	persona := getString(personaData, "persona")

	c, err := m.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch customer: %w", err)
	}

	c.Persona = persona
	if err := m.repo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to update persona: %w", err)
	}

	return map[string]any{"customer": c}, nil
}

// UpdateCustomerDetails updates the customer's profile information.
func (m *Module) UpdateCustomerDetails(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	customerID := getString(workflowInput, "id")
	c, err := m.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	if name := getString(workflowInput, "name"); name != "" {
		c.Name = name
	}
	if email := getString(workflowInput, "email"); email != "" {
		c.Email = email
	}

	if err := m.repo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return map[string]any{"customer": c}, nil
}

// CalculatePersonaStep wraps CalculatePersona to mdk.StepHandler.
func (m *Module) CalculatePersonaStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.CalculatePersona(sCtx.Ctx, map[string]any{
		"input": sCtx.Input,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// UpdatePersonaStep wraps UpdatePersona to mdk.StepHandler.
func (m *Module) UpdatePersonaStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.UpdatePersona(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// UpdateCustomerDetailsStep wraps UpdateCustomerDetails to mdk.StepHandler.
func (m *Module) UpdateCustomerDetailsStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.UpdateCustomerDetails(sCtx.Ctx, map[string]any{
		"input": sCtx.Input,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
