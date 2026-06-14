package cart

import (
	"context"
	"os"
	"testing"

	"github.com/GoHyperrr/mdk/mdktest"
)

func TestCartRepository(t *testing.T) {
	dbFile := "cart_repo_test.db"
	defer os.Remove(dbFile)

	database, _ := mdktest.SetupTestDB(dbFile)
	rt := mdktest.NewTestRuntime(database)

	mod := NewModule()
	_ = mod.Init(context.Background(), rt)
	_ = database.AutoMigrate(mod.Models()...)

	t.Run("GetActiveByCustomerID", func(t *testing.T) {
		c2 := &Cart{ID: "cart2", CustomerID: "cust2", Status: CartActive}
		mod.Repo().Save(context.Background(), c2)
		
		got, err := mod.Repo().GetActiveByCustomerID(context.Background(), "cust2")
		if err != nil || got.ID != "cart2" {
			t.Errorf("GetActiveByCustomerID failed: %v", err)
		}
	})

	t.Run("ClearItems", func(t *testing.T) {
		c2 := &Cart{ID: "cart3", CustomerID: "cust3", Status: CartActive}
		mod.Repo().Save(context.Background(), c2)
		
		c2.Items = append(c2.Items, CartItem{ID: "item-c3", CartID: "cart3", ProductID: "p1", Quantity: 1})
		mod.Repo().Save(context.Background(), c2)
		
		err := mod.Repo().ClearItems(context.Background(), "cart3")
		if err != nil {
			t.Fatalf("ClearItems failed: %v", err)
		}
		
		refreshed, _ := mod.Repo().GetByID(context.Background(), "cart3")
		if len(refreshed.Items) != 0 {
			t.Errorf("expected 0 items after clear, got %d", len(refreshed.Items))
		}
	})
}
