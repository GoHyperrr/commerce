package search

import (
	"context"
	"testing"

	"github.com/GoHyperrr/commerce/product"
	"github.com/GoHyperrr/mdk"
	"github.com/GoHyperrr/mdk/mdktest"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSearchModule(t *testing.T) {
	database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	rt := mdktest.NewTestRuntime(database)

	// Mock Product module
	prodMod := product.NewModule()
	_ = prodMod.Init(context.Background(), rt)

	mod := NewModule()
	_ = mod.Init(context.Background(), rt)
	mod.SetProductModule(prodMod)

	var models []any
	models = append(models, prodMod.Models()...)
	models = append(models, mod.Models()...)
	_ = database.AutoMigrate(models...)
	runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

	// Seed products
	prodMod.Repo().Save(context.Background(), &product.Product{
		ID:     "p1",
		Name:   "Go Gopher",
		Handle: "go-gopher",
		Variants: []product.ProductVariant{
			{ID: "v1", Title: "Default", Price: 10.0},
		},
	})
	prodMod.Repo().Save(context.Background(), &product.Product{
		ID:     "p2",
		Name:   "Rust Crab",
		Handle: "rust-crab",
		Variants: []product.ProductVariant{
			{ID: "v2", Title: "Default", Price: 15.0},
		},
	})

	t.Run("Search Success", func(t *testing.T) {
		wf := mdk.Workflow{
			ID:    "test-search-wf",
			Name:  "Test Search Products",
			Steps: []mdk.Step{{ID: "search_step", Uses: "search.product_catalog"}},
		}
		_ = runner.Register(wf)

		input := map[string]any{"query": "Go", "limit": 1.0}
		res, err := runner.ExecuteSync(context.Background(), "s1", "test-search-wf", input)
		if err != nil {
			t.Fatalf("workflow failed: %v", err)
		}

		results, ok := res["search"].([]*product.Product)
		if !ok {
			results = res["search_step"].([]*product.Product)
		}
		if len(results) != 1 || results[0].Name != "Go Gopher" {
			t.Errorf("unexpected results: %v", results)
		}
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		_, err := mod.SearchProducts(context.Background(), "string")
		if err == nil {
			t.Error("expected error for invalid input type")
		}

		_, err = mod.SearchProducts(context.Background(), map[string]any{"wrong": 1})
		if err == nil {
			t.Error("expected error for missing workflow input")
		}

		mNoProd := NewModule()
		mNoProdDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		mNoProdRt := mdktest.NewTestRuntime(mNoProdDB)
		_ = mNoProd.Init(context.Background(), mNoProdRt)
		_, err = mNoProd.SearchProducts(context.Background(), map[string]any{"input": map[string]any{"query": "x"}})
		if err == nil {
			t.Error("expected error for missing product module")
		}
	})
}
