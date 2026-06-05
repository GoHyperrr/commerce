package cart

import (
	"context"
	"testing"

	"github.com/GoHyperrr/mdk/mdktest"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCartWorkflow(t *testing.T) {
	t.Run("Add Item Workflow", func(t *testing.T) {
		database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		rt := mdktest.NewTestRuntime(database)

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = database.AutoMigrate(mod.Models()...)
		runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

		c := &Cart{ID: "cart1", CustomerID: "cust1", Status: CartActive}
		mod.Repo().Save(context.Background(), c)

		input := map[string]any{"cart_id": "cart1", "product_id": "prod1", "quantity": 2, "price": 25.0}

		res, err := runner.ExecuteSync(context.Background(), "add_1", "cart.add", input)
		if err != nil { t.Fatalf("workflow failed: %v", err) }

		resMap := res["add"].(map[string]any)
		updated := resMap["cart"].(*Cart)
		if len(updated.Items) != 1 { t.Errorf("expected 1 item, got %d", len(updated.Items)) }
	})

	t.Run("Remove Item Workflow", func(t *testing.T) {
		database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		rt := mdktest.NewTestRuntime(database)

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = database.AutoMigrate(mod.Models()...)
		runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

		c := &Cart{ID: "cart1", Items: []CartItem{{ID: "i1", CartID: "cart1", ProductID: "p1", Quantity: 1}}, Status: CartActive}
		mod.Repo().Save(context.Background(), c)

		input := map[string]any{"cart_id": "cart1", "item_id": "i1"}

		res, err := runner.ExecuteSync(context.Background(), "remove_1", "cart.remove", input)
		if err != nil { t.Fatalf("workflow failed: %v", err) }

		resMap := res["remove"].(map[string]any)
		updated := resMap["cart"].(*Cart)
		if len(updated.Items) != 0 { t.Errorf("expected 0 items, got %d", len(updated.Items)) }
	})

	t.Run("Checkout Workflow", func(t *testing.T) {
		database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		rt := mdktest.NewTestRuntime(database)

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = database.AutoMigrate(mod.Models()...)
		runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

		c := &Cart{ID: "cart1", Items: []CartItem{{ID: "i1", CartID: "cart1", ProductID: "p1", Quantity: 1}}, Status: CartActive}
		mod.Repo().Save(context.Background(), c)

		_, err := runner.ExecuteSync(context.Background(), "checkout_1", "cart.checkout", map[string]any{"cart_id": "cart1"})
		if err != nil { t.Fatalf("workflow failed: %v", err) }

		final, _ := mod.Repo().GetByID(context.Background(), "cart1")
		if final.Status != CartCompleted { t.Errorf("expected COMPLETED status, got %s", final.Status) }
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		rt := mdktest.NewTestRuntime(database)

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = database.AutoMigrate(mod.Models()...)

		// 1. AddItem - Invalid Input
		_, err := mod.AddItem(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }

		_, err = mod.AddItem(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		// 2. AddItem - Cart Not Active
		c := &Cart{ID: "c-inactive", Status: CartCompleted}
		mod.Repo().Save(context.Background(), c)
		_, err = mod.AddItem(context.Background(), map[string]any{"input": map[string]any{"cart_id": "c-inactive"}})
		if err == nil { t.Error("expected error for inactive cart") }
		
		_, err = mod.AddItem(context.Background(), map[string]any{"input": "invalid"})
		if err == nil { t.Error("expected error for invalid input format") }

		// 3. RemoveItem - Invalid Input
		_, err = mod.RemoveItem(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }

		_, err = mod.RemoveItem(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }
		
		_, err = mod.RemoveItem(context.Background(), map[string]any{"input": map[string]any{"cart_id": "ghost"}})
		if err == nil { t.Error("expected error for non-existent cart remove") }

		// 4. Checkout - Empty Cart
		c2 := &Cart{ID: "c-empty", Status: CartActive}
		mod.Repo().Save(context.Background(), c2)
		_, err = mod.Checkout(context.Background(), map[string]any{"input": map[string]any{"cart_id": "c-empty"}})
		if err == nil { t.Error("expected error for empty cart") }
		
		_, err = mod.Checkout(context.Background(), map[string]any{"input": map[string]any{"cart_id": "ghost"}})
		if err == nil { t.Error("expected error for non-existent cart checkout") }
	})
}
