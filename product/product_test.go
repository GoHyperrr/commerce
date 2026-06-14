package product

import (
	"context"
	"testing"

	"github.com/GoHyperrr/mdk"
	"github.com/GoHyperrr/mdk/mdktest"
)

func TestProductWorkflow(t *testing.T) {
	rt, _ := mdktest.NewInMemoryTestRuntime()

	mod := NewModule()
	if err := mod.Init(context.Background(), rt); err != nil {
		t.Fatalf("failed to init module: %v", err)
	}

	_ = rt.DB().AutoMigrate(mod.Models()...)
	runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

	t.Run("Create Product Workflow", func(t *testing.T) {
		input := map[string]any{
			"id":          "p1",
			"name":        "Test Product",
			"handle":      "test-product",
			"description": "Desc",
			"variants": []any{
				map[string]any{"title": "Default", "price": 100.0},
			},
		}

		res, err := runner.ExecuteSync(context.Background(), "create_1", "product.create", input)
		if err != nil {
			t.Fatalf("workflow failed: %v", err)
		}

		resMap := res["persist"].(map[string]any)
		p, ok := resMap["product"].(*Product)
		if !ok || p.Name != "Test Product" || p.Handle != "test-product" {
			t.Errorf("expected Test Product, got %v", res["persist"])
		}

		// Verify preloaded relations
		fetched, err := mod.Repo().GetByID(context.Background(), "p1")
		if err != nil {
			t.Fatalf("failed to get product by ID: %v", err)
		}
		if len(fetched.Variants) != 1 || fetched.Variants[0].Price != 100.0 {
			t.Errorf("unexpected variant: %+v", fetched.Variants)
		}
	})

	t.Run("Invalid Product", func(t *testing.T) {
		wf := mdk.Workflow{
			ID:    "test-invalid-wf",
			Name:  "Test Invalid Product",
			Steps: []mdk.Step{{ID: "v1", Uses: "product.validate_product"}},
		}
		_ = runner.Register(wf)

		input := map[string]any{
			"name":   "",
			"handle": "",
		}

		_, err := runner.ExecuteSync(context.Background(), "create_invalid", "test-invalid-wf", input)
		if err == nil {
			t.Error("expected validation error")
		}
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		rt, _ := mdktest.NewInMemoryTestRuntime()

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = rt.DB().AutoMigrate(mod.Models()...)

		// 1. ValidateProduct - Invalid Input
		_, err := mod.ValidateProduct(context.Background(), "string")
		if err == nil {
			t.Error("expected error for invalid input type")
		}
		_, err = mod.ValidateProduct(context.Background(), map[string]any{"wrong": 1})
		if err == nil {
			t.Error("expected error for missing workflow input")
		}

		// 2. PersistProduct - Invalid Input
		_, err = mod.PersistProduct(context.Background(), "string")
		if err == nil {
			t.Error("expected error for invalid input type")
		}
		_, err = mod.PersistProduct(context.Background(), map[string]any{"wrong": 1})
		if err == nil {
			t.Error("expected error for missing validation results")
		}

		// 3. UpdateProductDetails - Invalid Input
		_, err = mod.UpdateProductDetails(context.Background(), "string")
		if err == nil {
			t.Error("expected error for invalid input type")
		}
		_, err = mod.UpdateProductDetails(context.Background(), map[string]any{"wrong": 1})
		if err == nil {
			t.Error("expected error for missing workflow input")
		}

		_, err = mod.UpdateProductDetails(context.Background(), map[string]any{"input": "invalid"})
		if err == nil {
			t.Error("expected error for invalid input format")
		}

		// 4. UpdateProductDetails - Product Not Found
		_, err = mod.UpdateProductDetails(context.Background(), map[string]any{"input": map[string]any{"id": "ghost"}})
		if err == nil {
			t.Error("expected error for non-existent product")
		}

		// 5. PersistProduct - Save Failure
		badDB, _ := mdktest.SetupTestDB("fail_save.db")
		sqlDB, _ := badDB.DB()
		sqlDB.Close()

		mod.repo = NewRepository(badDB)
		failInput := map[string]any{
			"validate": map[string]any{
				"id": "p_fail", "name": "Fail", "handle": "fail", "description": "",
			},
		}
		_, err = mod.PersistProduct(context.Background(), failInput)
		if err == nil {
			t.Error("expected error for failed save in PersistProduct")
		}

		// 6. UpdateProductDetails - Success (All fields)
		mod.repo = NewRepository(rt.DB()) // Ensure we use the good DB
		p := &Product{
			ID:          "p_update",
			Name:        "Old Name",
			Handle:      "old-name",
			Description: "Old Desc",
			Variants: []ProductVariant{
				{ID: "v_update", Title: "Default", Price: 50.0},
			},
		}
		mod.Repo().Save(context.Background(), p)

		updateAllInput := map[string]any{
			"input": map[string]any{
				"id":          "p_update",
				"name":        "New Name",
				"handle":      "new-name",
				"description": "New Desc",
			},
		}
		_, err = mod.UpdateProductDetails(context.Background(), updateAllInput)
		if err != nil {
			t.Errorf("UpdateProductDetails failed: %v", err)
		}
		updated, _ := mod.repo.GetByID(context.Background(), "p_update")
		if updated.Name != "New Name" || updated.Description != "New Desc" || updated.Handle != "new-name" {
			t.Errorf("UpdateProductDetails did not update all fields: %+v", updated)
		}

		// 7. UpdateProductDetails - Save Failure
		mod.repo = NewRepository(badDB)
		_, err = mod.UpdateProductDetails(context.Background(), updateAllInput)
		if err == nil {
			t.Error("expected error for failed save in UpdateProductDetails")
		}
		mod.repo = NewRepository(rt.DB()) // Restore to good DB

		// 8. PersistProduct - Invalid validated data format
		_, err = mod.PersistProduct(context.Background(), map[string]any{"validate": "not-a-map"})
		if err == nil {
			t.Error("expected error for invalid validated data format")
		}
	})
}
