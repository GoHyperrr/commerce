package search

import (
	"context"
	"os"
	"testing"

	"github.com/GoHyperrr/commerce/product"
	"github.com/GoHyperrr/hyperrr/pkg/workflow"
	"github.com/GoHyperrr/hyperrr/pkg/config"
	"github.com/GoHyperrr/hyperrr/pkg/db"
	"github.com/GoHyperrr/hyperrr/pkg/eventbus"
	"github.com/GoHyperrr/hyperrr/pkg/registry"
	"github.com/GoHyperrr/mdk"
)

func TestSearchModule(t *testing.T) {
	dbFile := "search_test.db"
	defer os.Remove(dbFile)

	cfg := &config.Config{DBDriver: "sqlite", DBDSN: dbFile}
	database, _ := db.Connect(cfg)
	bus := eventbus.NewInMemBus()
	runner := workflow.NewRunner(bus, nil, nil)

	// Mock Product module
	prodMod := product.NewModule()
	prodMod.Init(context.Background(), registry.NewRuntime(&registry.Dependencies{DB: database, EventBus: bus, Runner: runner}))
	db.Register(prodMod.Models()...)
	
	mod := NewModule()
	mod.Init(context.Background(), registry.NewRuntime(&registry.Dependencies{DB: database, EventBus: bus, Runner: runner}))
	mod.SetProductModule(prodMod)
	db.Register(mod.Models()...)
	database.AutoMigrateAll()

	// Seed products
	prodMod.Repo().Save(context.Background(), &product.Product{ID: "p1", Name: "Go Gopher", Price: 10.0})
	prodMod.Repo().Save(context.Background(), &product.Product{ID: "p2", Name: "Rust Crab", Price: 15.0})

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

		results := res["search"].([]*product.Product)
		if len(results) != 1 || results[0].Name != "Go Gopher" {
			t.Errorf("unexpected results: %v", results)
		}
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		_, err := mod.SearchProducts(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		
		_, err = mod.SearchProducts(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		mNoProd := NewModule()
		mNoProdBus := eventbus.NewInMemBus()
		mNoProdRunner := workflow.NewRunner(mNoProdBus, nil, nil)
		mNoProd.Init(context.Background(), registry.NewRuntime(&registry.Dependencies{DB: database, EventBus: mNoProdBus, Runner: mNoProdRunner}))
		_, err = mNoProd.SearchProducts(context.Background(), map[string]any{"input": map[string]any{"query": "x"}})
		if err == nil { t.Error("expected error for missing product module") }
	})
}
