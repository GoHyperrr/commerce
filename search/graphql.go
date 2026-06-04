package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/commerce/product"
	"github.com/google/uuid"
)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"searchProducts": m.SearchProductsResolver,
	}
}

func (m *Module) Mutations() map[string]any {
	return nil
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) SearchProductsResolver(ctx context.Context, query string, limit *int) ([]*product.Product, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	lim := 10.0
	if limit != nil {
		lim = float64(*limit)
	}

	workflowInput := map[string]any{
		"query": query,
		"limit": lim,
	}

	execID := "search_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "search.products", workflowInput)
	if err != nil {
		return nil, err
	}

	var prodsRaw []*product.Product
	if err := decodeResult(results["search"], &prodsRaw); err != nil {
		return nil, fmt.Errorf("failed to retrieve search results from workflow: %w", err)
	}

	return prodsRaw, nil
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
