package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/commerce/product"
	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

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

func (m *Module) SearchProductsResolver(ctx context.Context, query string, limit *int) ([]*model.Product, error) {
	wf, err := m.deps.Registry.Get("search.products")
	if err != nil {
		return nil, err
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
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	var prodsRaw []*product.Product
	if err := decodeResult(results["search"], &prodsRaw); err != nil {
		return nil, fmt.Errorf("failed to retrieve search results from workflow: %w", err)
	}

	res := make([]*model.Product, 0, len(prodsRaw))
	for _, p := range prodsRaw {
		res = append(res, mapProductToModel(p))
	}
	return res, nil
}

func mapProductToModel(p *product.Product) *model.Product {
	return &model.Product{
		ID:          p.ID,
		Name:        p.Name,
		Description: &p.Description,
		Price:       p.Price,
		Currency:    p.Currency,
	}
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
