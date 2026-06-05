package customer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/GoHyperrr/mdk"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCustomerWorkflow(t *testing.T) {
	database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	rt := mdk.NewTestRuntime(database)

	testProj := &mdk.TestProjector{}
	ctxMod := &mdk.TestContextModule{Proj: testProj}
	rt.SetModule("core.context", ctxMod)

	mod := NewModule()
	mod.SetProjector(testProj)

	if err := mod.Init(context.Background(), rt); err != nil {
		t.Fatalf("failed to init module: %v", err)
	}

	_ = database.AutoMigrate(mod.Models()...)
	runner := rt.Workflows().(*mdk.TestWorkflowEngine)

	t.Run("Segmentation Workflow", func(t *testing.T) {
		// Create a customer first
		c := &Customer{ID: "c1", Name: "John Doe", Email: "john@example.com"}
		mod.Repo().Save(context.Background(), c)

		// Seed lineages to get WHALE persona (needs > 5 orders)
		for i := 0; i < 6; i++ {
			wfID := fmt.Sprintf("wf_%d", i)
			testProj.Lineages = append(testProj.Lineages, mdk.TestLineageData{
				ID:    wfID,
				Name:  "fulfillment.v1",
				State: "COMPLETED",
			})
		}

		input := map[string]any{
			"customer_id": "c1",
			"order_total": 1500.0,
		}

		_, err := runner.ExecuteSync(context.Background(), "seg_1", "customer.segmentation", input)
		if err != nil {
			t.Fatalf("workflow failed: %v", err)
		}

		// Verify Persona update
		updated, _ := mod.Repo().GetByID(context.Background(), "c1")
		if updated.Persona != "WHALE" {
			t.Errorf("expected WHALE persona, got %s", updated.Persona)
		}
	})

	t.Run("Get and List", func(t *testing.T) {
		c, err := mod.Repo().GetByID(context.Background(), "c1")
		if err != nil || c.Name != "John Doe" {
			t.Error("GetByID failed")
		}

		c2, err := mod.Repo().GetByUserID(context.Background(), "u123")
		if err == nil {
			t.Error("expected error for non-existent user_id")
		}
		if c2 != nil {
			t.Error("expected nil customer for non-existent user_id")
		}
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		rt := mdk.NewTestRuntime(database)

		mod := NewModule()
		_ = mod.Init(context.Background(), rt)
		_ = database.AutoMigrate(mod.Models()...)

		// 1. CalculatePersona - Invalid Input
		_, err := mod.CalculatePersona(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.CalculatePersona(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		// 2. UpdatePersona - Invalid Input
		_, err = mod.UpdatePersona(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.UpdatePersona(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing persona data") }

		// 3. UpdateCustomerDetails - Invalid Input
		_, err = mod.UpdateCustomerDetails(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.UpdateCustomerDetails(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		// 4. UpdateCustomerDetails - Customer Not Found
		_, err = mod.UpdateCustomerDetails(context.Background(), map[string]any{"input": map[string]any{"id": "ghost"}})
		if err == nil { t.Error("expected error for non-existent customer") }
		
		// 5. identity.user_created - Missing actor_id (should skip gracefully)
		rt.Bus().Publish(context.Background(), mdk.Event{
			Namespace: "identity",
			Type:      "user_created",
			Payload:   map[string]any{"user_id": "u_no_actor", "email": "test@test.com"},
		})
		_, err = mod.Repo().GetByUserID(context.Background(), "u_no_actor")
		if err == nil {
			t.Error("expected no customer to be created for missing actor_id")
		}

		// 6. CalculatePersona - Nil brain
		badMod := &Module{repo: mod.repo}
		_, err = badMod.CalculatePersona(context.Background(), map[string]any{"input": map[string]any{"customer_id": "c1"}})
		if err == nil || !strings.Contains(err.Error(), "ML brain not initialized") {
			t.Errorf("expected ML brain not initialized error, got %v", err)
		}

		// 7. order.completed - Missing customer_id
		rt.Bus().Publish(context.Background(), mdk.Event{
			Namespace: "order",
			Type:      "completed",
			Payload:   map[string]any{"wrong": "data"},
		})
		// Should just skip gracefully

		// 8. GetByUserID - Success
		c := &Customer{ID: "c_user", UserID: "u_real", Name: "Real User"}
		mod.Repo().Save(context.Background(), c)
		got, err := mod.Repo().GetByUserID(context.Background(), "u_real")
		if err != nil || got.ID != "c_user" {
			t.Errorf("GetByUserID failed: %v", err)
		}

		// 9. UpdateCustomerDetails - Success
		updateInput := map[string]any{
			"input": map[string]any{
				"id":    "c_user",
				"name":  "Updated Name",
				"email": "updated@example.com",
			},
		}
		_, err = mod.UpdateCustomerDetails(context.Background(), updateInput)
		if err != nil {
			t.Errorf("UpdateCustomerDetails failed: %v", err)
		}
		updated, _ := mod.Repo().GetByID(context.Background(), "c_user")
		if updated.Name != "Updated Name" || updated.Email != "updated@example.com" {
			t.Error("UpdateCustomerDetails did not save changes")
		}

		// 10. UpdateCustomerDetails - Save Failure
		badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		sqlDB, _ := badDB.DB()
		sqlDB.Close()

		originalRepo := mod.repo
		mod.repo = NewRepository(badDB)
		_, err = mod.UpdateCustomerDetails(context.Background(), updateInput)
		if err == nil { t.Error("expected error for failed save in UpdateCustomerDetails") }
		
		// 11. UpdatePersona - Customer Not Found
		mod.repo = originalRepo // Restore to find "c_user" if needed, but here we want non-existent
		personaInputGhost := map[string]any{
			"calculate": map[string]any{
				"customer_id": "ghost_cust",
				"persona":     "WHALE",
			},
		}
		_, err = mod.UpdatePersona(context.Background(), personaInputGhost)
		if err == nil { t.Error("expected error for non-existent customer in UpdatePersona") }

		// 12. UpdatePersona - Save Failure
		mod.repo = NewRepository(badDB)
		personaInputValid := map[string]any{
			"calculate": map[string]any{
				"customer_id": "c_user",
				"persona":     "WHALE",
			},
		}
		_, err = mod.UpdatePersona(context.Background(), personaInputValid)
		if err == nil { t.Error("expected error for failed save in UpdatePersona") }
		
		mod.repo = originalRepo

		// 13. UpdatePersona - Fallback Step Name
		personaInputFallback := map[string]any{
			"customer.calculate_persona": map[string]any{
				"customer_id": "c_user",
				"persona":     "GOLD",
			},
		}
		_, err = mod.UpdatePersona(context.Background(), personaInputFallback)
		if err != nil { t.Errorf("UpdatePersona fallback failed: %v", err) }
		updated, _ = mod.repo.GetByID(context.Background(), "c_user")
		if updated.Persona != "GOLD" { t.Errorf("expected GOLD, got %s", updated.Persona) }

		// 14. UpdateCustomerDetails - Empty fields (no change)
		emptyInput := map[string]any{
			"input": map[string]any{
				"id":    "c_user",
				"name":  "",
				"email": "",
			},
		}
		_, err = mod.UpdateCustomerDetails(context.Background(), emptyInput)
		if err != nil { t.Errorf("UpdateCustomerDetails failed: %v", err) }
		updated, _ = mod.repo.GetByID(context.Background(), "c_user")
		if updated.Name != "Updated Name" || updated.Email != "updated@example.com" {
			t.Error("UpdateCustomerDetails changed fields that were empty in input")
		}

		// 15. identity.user_created - Success
		rt.Bus().Publish(context.Background(), mdk.Event{
			Namespace: "identity",
			Type:      "user_created",
			Payload: map[string]any{
				"actor_id": "u_new",
				"user_id":  "u_new_id",
				"name":     "New User",
				"email":    "new@test.com",
			},
		})
		got, err = mod.Repo().GetByUserID(context.Background(), "u_new")
		if err != nil || got.Name != "New User" {
			t.Errorf("identity.user_created success handler failed: %v", err)
		}

		// 16. order.completed - Success (Triggers background workflow)
		rt.Bus().Publish(context.Background(), mdk.Event{
			Namespace: "order",
			Type:      "completed",
			Payload:   map[string]any{"customer_id": "c_user"},
		})
	})

	t.Run("Handler Error Paths Surgical", func(t *testing.T) {
		ctx := context.Background()
		// 1. CalculatePersona - Invalid Input
		_, err := mod.CalculatePersona(ctx, "string")
		if err == nil { t.Error("expected error for invalid input type") }
		
		_, err = mod.CalculatePersona(ctx, map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		// 2. UpdatePersona - Missing Data
		_, err = mod.UpdatePersona(ctx, map[string]any{})
		if err == nil { t.Error("expected error for missing persona data") }
	})
}
