package product

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getProduct":            m.GetProduct,
		"listProducts":          m.ListProducts,
		"getProductsByTaxonomy": m.GetProductsByTaxonomy,
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

type syncExecutor interface {
	ExecuteSync(ctx context.Context, id string, workflowID string, input map[string]any) (map[string]any, error)
}

func (m *Module) CreateProduct(ctx context.Context, input CreateProductInput) (*Product, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	// Convert input struct to map[string]any for workflow context
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var workflowInput map[string]any
	if err := json.Unmarshal(inputBytes, &workflowInput); err != nil {
		return nil, err
	}

	execID := "create_prod_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "product.create", workflowInput)
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

	return &domainRes, nil
}

func (m *Module) UpdateProduct(ctx context.Context, id string, input UpdateProductInput) (*Product, error) {
	executor, ok := m.rt.Workflows().(syncExecutor)
	if !ok {
		return nil, fmt.Errorf("workflow engine does not support synchronous execution")
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var workflowInput map[string]any
	if err := json.Unmarshal(inputBytes, &workflowInput); err != nil {
		return nil, err
	}
	workflowInput["id"] = id

	execID := "update_prod_" + uuid.New().String()
	results, err := executor.ExecuteSync(ctx, execID, "product.update", workflowInput)
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

	return &domainRes, nil
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

func (m *Module) GetProduct(ctx context.Context, id string) (*Product, error) {
	return m.repo.GetByID(ctx, id)
}

func (m *Module) ListProducts(ctx context.Context) ([]*Product, error) {
	return m.repo.List(ctx)
}

func (m *Module) GetProductsByTaxonomy(ctx context.Context, termID string) ([]*Product, error) {
	var productIDs []string
	err := m.rt.DB().WithContext(ctx).Table("taxonomy_relations").
		Where("term_id = ? AND resource_type = ?", termID, "product").
		Pluck("resource_id", &productIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query taxonomy relations: %w", err)
	}

	if len(productIDs) == 0 {
		return []*Product{}, nil
	}

	var products []*Product
	err = m.rt.DB().WithContext(ctx).
		Preload("Options").
		Preload("Variants").
		Preload("Variants.Options").
		Preload("Images").
		Where("id IN ?", productIDs).
		Find(&products).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products by IDs: %w", err)
	}

	return products, nil
}

func decodeResult(src any, dest any) error {
	dataBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, dest)
}
