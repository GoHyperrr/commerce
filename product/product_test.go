package product

import (
	"context"
	"os"
	"testing"

	"github.com/GoHyperrr/mdk"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestProductWorkflow(t *testing.T) {
	dbFile := "product_test.db"
	defer os.Remove(dbFile)

	database, _ := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	rt := mdk.NewTestRuntime(database)

	mod := NewModule()
	if err := mod.Init(context.Background(), rt); err != nil {
		t.Fatalf("failed to init module: %v", err)
	}

	_ = database.AutoMigrate(mod.Models()...)
	runner := rt.Workflows().(*mdk.TestWorkflowEngine)

	t.Run("Create Product Workflow", func(t *testing.T) {
		input := map[string]any{
			"id":          "p1",
			"name":        "Test Product",
			"description": "Desc",
			"price":       100.0,
		}

		res, err := runner.ExecuteSync(context.Background(), "create_1", "product.create", input)
		if err != nil {
			t.Fatalf("workflow failed: %v", err)
		}

		resMap := res["persist"].(map[string]any)
		p, ok := resMap["product"].(*Product)
		if !ok || p.Name != "Test Product" {
			t.Errorf("expected Test Product, got %v", res["persist"])
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
			"name":  "",
			"price": -10.0,
		}

		_, err := runner.ExecuteSync(context.Background(), "create_invalid", "test-invalid-wf", input)
		if err == nil {
			t.Error("expected validation error")
		}
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		dbFile := "prod_err_test.db"
		defer os.Remove(dbFile)
		database, _ := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
		rt := mdk.NewTestRuntime(database)

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = database.AutoMigrate(mod.Models()...)

		// 1. ValidateProduct - Invalid Input
		_, err := mod.ValidateProduct(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.ValidateProduct(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		// 2. PersistProduct - Invalid Input
		_, err = mod.PersistProduct(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.PersistProduct(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing validation results") }

		// 3. UpdateProductDetails - Invalid Input
		_, err = mod.UpdateProductDetails(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.UpdateProductDetails(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }
		
		_, err = mod.UpdateProductDetails(context.Background(), map[string]any{"input": "invalid"})
		if err == nil { t.Error("expected error for invalid input format") }

		// 4. UpdateProductDetails - Product Not Found
		_, err = mod.UpdateProductDetails(context.Background(), map[string]any{"input": map[string]any{"id": "ghost"}})
		if err == nil { t.Error("expected error for non-existent product") }

		// 5. PersistProduct - Save Failure
		badDB, _ := gorm.Open(sqlite.Open("fail_save.db"), &gorm.Config{})
		sqlDB, _ := badDB.DB()
		sqlDB.Close()
		
		mod.repo = NewRepository(badDB)
		failInput := map[string]any{
			"validate": map[string]any{
				"id": "p_fail", "name": "Fail", "description": "", "price": 10.0,
			},
		}
		_, err = mod.PersistProduct(context.Background(), failInput)
		if err == nil { t.Error("expected error for failed save in PersistProduct") }

		// 6. UpdateProductDetails - Success (All fields)
		mod.repo = NewRepository(database) // Ensure we use the good DB
		p := &Product{ID: "p_update", Name: "Old Name", Description: "Old Desc", Price: 50.0}
		database.Save(p)
		updateAllInput := map[string]any{
			"input": map[string]any{
				"id":          "p_update",
				"name":        "New Name",
				"description": "New Desc",
				"price":       75.0,
			},
		}
		_, err = mod.UpdateProductDetails(context.Background(), updateAllInput)
		if err != nil {
			t.Errorf("UpdateProductDetails failed: %v", err)
		}
		updated, _ := mod.repo.GetByID(context.Background(), "p_update")
		if updated.Name != "New Name" || updated.Description != "New Desc" || updated.Price != 75.0 {
			t.Errorf("UpdateProductDetails did not update all fields: %+v", updated)
		}

		// 7. UpdateProductDetails - Save Failure
		mod.repo = NewRepository(badDB)
		_, err = mod.UpdateProductDetails(context.Background(), updateAllInput)
		if err == nil { t.Error("expected error for failed save in UpdateProductDetails") }
		mod.repo = NewRepository(database) // Restore to good DB

		// 8. UpdateProductDetails - Int price
		updateIntInput := map[string]any{
			"input": map[string]any{
				"id":    "p_update",
				"price": 99,
			},
		}
		_, err = mod.UpdateProductDetails(context.Background(), updateIntInput)
		if err != nil { t.Errorf("UpdateProductDetails failed with int price: %v", err) }
		updated, _ = mod.repo.GetByID(context.Background(), "p_update")
		if updated.Price != 99.0 { t.Errorf("expected 99.0, got %v", updated.Price) }

		// 9. PersistProduct - Invalid validated data format
		_, err = mod.PersistProduct(context.Background(), map[string]any{"validate": "not-a-map"})
		if err == nil { t.Error("expected error for invalid validated data format") }

		os.Remove("fail_save.db")
	})
}
