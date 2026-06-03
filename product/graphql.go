package product

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/hyperrr/api/graph/model"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/google/uuid"
)

// Ensure Module implements registry.GraphQLProvider at compile time.
var _ registry.GraphQLProvider = (*Module)(nil)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getProduct":   m.GetProduct,
		"listProducts": m.ListProducts,
	}
}

func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"createProduct": m.CreateProduct,
		"updateProduct": m.UpdateProduct,
		"deleteProduct": m.DeleteProduct,
	}
}

func (m *Module) FieldResolvers() map[string]any {
	return nil
}

func (m *Module) CreateProduct(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
	wf, err := m.deps.Registry.Get("product.create")
	if err != nil {
		return nil, err
	}

	desc := ""
	if input.Description != nil {
		desc = *input.Description
	}

	workflowInput := map[string]any{
		"id":          input.ID,
		"name": input.Name,
		"description": desc,
		"price":       input.Price,
	}

	execID := "create_prod_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["persist"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve created product from workflow results")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from persist step")
	}

	var domainRes Product
	if err := decodeResult(resMap["product"], &domainRes); err != nil {
		return nil, fmt.Errorf("invalid product type in results: %w", err)
	}

	return mapProductToModel(&domainRes), nil
}

func (m *Module) UpdateProduct(ctx context.Context, id string, input model.UpdateProductInput) (*model.Product, error) {
	wf, err := m.deps.Registry.Get("product.update")
	if err != nil {
		return nil, err
	}

	workflowInput := map[string]any{
		"id": id,
	}
	if input.Name != nil {
		workflowInput["name"] = *input.Name
	}
	if input.Description != nil {
		workflowInput["description"] = *input.Description
	}
	if input.Price != nil {
		workflowInput["price"] = *input.Price
	}

	execID := "update_prod_" + uuid.New().String()
	results, err := m.deps.Runner.Execute(ctx, execID, wf, workflowInput)
	if err != nil {
		return nil, err
	}

	resRaw, ok := results["update"]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve updated product from workflow results")
	}

	resMap, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid result format from update step")
	}

	var domainRes Product
	if err := decodeResult(resMap["product"], &domainRes); err != nil {
		return nil, fmt.Errorf("invalid product type in results: %w", err)
	}

	return mapProductToModel(&domainRes), nil
}

func (m *Module) DeleteProduct(ctx context.Context, id string) (bool, error) {
	p, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("product not found")
	}

	err = m.repo.Delete(ctx, p.ID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *Module) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	p, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return mapProductToModel(p), nil
}

func (m *Module) ListProducts(ctx context.Context) ([]*model.Product, error) {
	products, err := m.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Product, 0, len(products))
	for _, p := range products {
		res = append(res, mapProductToModel(p))
	}
	return res, nil
}

func mapProductToModel(p *Product) *model.Product {
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
