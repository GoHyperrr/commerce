package product

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for the Product.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.product"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "product.create",
		Name:        "product.create",
		Description: "Create a new product in the catalog.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":          map[string]any{"type": "string"},
				"name":        map[string]any{"type": "string"},
				"handle":      map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"status":      map[string]any{"type": "string", "enum": []string{"ACTIVE", "DRAFT", "ARCHIVED"}},
				"type":        map[string]any{"type": "string", "enum": []string{"PHYSICAL", "DIGITAL"}},
			},
			"required": []string{"name", "handle"},
		},
		Steps: []mdk.Step{
			{ID: "validate", Name: "Validate Product", Uses: "product.validate_product"},
			{ID: "persist", Name: "Persist Product", Uses: "product.persist_product", DependsOn: []string{"validate"}},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "product.update",
		Name:        "product.update",
		Description: "Update properties of an existing product in the catalog.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":          map[string]any{"type": "string"},
				"name":        map[string]any{"type": "string"},
				"handle":      map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"status":      map[string]any{"type": "string", "enum": []string{"ACTIVE", "DRAFT", "ARCHIVED"}},
			},
			"required": []string{"id"},
		},
		Steps: []mdk.Step{
			{ID: "update", Name: "Update Details", Uses: "product.update_details"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "product.get_by_taxonomy",
		Name:        "product.get_by_taxonomy",
		Description: "Retrieve a list of products linked to a specific taxonomy term ID.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"termId": map[string]any{"type": "string"},
			},
			"required": []string{"termId"},
		},
		Steps: []mdk.Step{
			{ID: "get_by_taxonomy", Name: "Retrieve Products", Uses: "product.get_by_taxonomy_step"},
		},
	})

	// Register workflow step handlers
	_ = rt.Workflows().RegisterHandler("product.validate_product", m.ValidateProductStep)
	_ = rt.Workflows().RegisterHandler("product.persist_product", m.PersistProductStep)
	_ = rt.Workflows().RegisterHandler("product.update_details", m.UpdateProductDetailsStep)
	_ = rt.Workflows().RegisterHandler("product.get_by_taxonomy_step", m.GetProductsByTaxonomyStep)

	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	products, err := m.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var res []mdk.MCPResource
	for _, p := range products {
		res = append(res, mdk.MCPResource{
			URI:         "product://" + p.ID,
			Name:        "Product: " + p.Name,
			Description: "Catalog details for product " + p.ID + " (" + p.Handle + ")",
			MimeType:    "application/json",
		})
	}
	return res, nil
}

func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	var productID string
	if n, err := fmt.Sscanf(uri, "product://%s", &productID); err != nil || n != 1 {
		return "", fmt.Errorf("invalid product URI")
	}
	p, err := m.repo.GetByID(ctx, productID)
	if err != nil {
		return "", err
	}
	dataBytes, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(dataBytes), nil
}

func (m *Module) GetProductsByTaxonomyStep(sCtx mdk.StepContext) mdk.StepResult {
	termID, _ := sCtx.Input["termId"].(string)
	if termID == "" {
		return mdk.StepResult{Err: fmt.Errorf("termId required")}
	}
	products, err := m.GetProductsByTaxonomy(sCtx.Ctx, termID)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"products": products}}
}

func (m *Module) Models() []any {
	return []any{&Product{}, &ProductOption{}, &ProductVariant{}, &VariantOption{}, &ProductImage{}}
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) Repo() *Repository {
	return m.repo
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
