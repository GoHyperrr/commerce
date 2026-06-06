package notification

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/GoHyperrr/mdk"
	"github.com/GoHyperrr/mdk/mdktest"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNotificationModule(t *testing.T) {
	database, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	rt := mdktest.NewTestRuntime(database)
	
	// Create mock provider
	mockProv := &MockProvider{}

	mod := NewModule(mockProv)
	_ = mod.Init(context.Background(), rt)
	_ = database.AutoMigrate(mod.Models()...)
	runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

	t.Run("Send Notification Success", func(t *testing.T) {
		recipient := fmt.Sprintf("test_%s@example.com", uuid.New().String()[:8])
		wf := mdk.Workflow{
			ID:    "test-send-wf",
			Name:  "Test Send Notification",
			Steps: []mdk.Step{{ID: "send", Uses: "notification.send"}},
		}
		_ = runner.Register(wf)

		input := map[string]any{
			"recipient": recipient,
			"channel":   "EMAIL",
			"subject":   "Test",
			"body":      "Hello",
		}

		res, err := runner.ExecuteSync(context.Background(), "n1", "test-send-wf", input)
		if err != nil {
			t.Fatalf("workflow failed: %v", err)
		}

		sendRes, ok := res["send"].(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", res["send"])
		}
		n, ok := sendRes["notification"].(*Notification)
		if !ok {
			t.Fatalf("expected notification pointer, got %T", sendRes["notification"])
		}
		if n.Status != StatusSent {
			t.Errorf("expected SENT status, got %s", n.Status)
		}

		// Verify Repo
		list, _ := mod.Repo().List(context.Background(), recipient)
		if len(list) != 1 {
			t.Error("expected 1 notification in repo")
		}
	})

	t.Run("Send Notification Failure", func(t *testing.T) {
		recipient := fmt.Sprintf("fail_%s@example.com", uuid.New().String()[:8])
		mockProv.ShouldFail = true
		
		wf := mdk.Workflow{
			ID:    "test-fail-wf",
			Name:  "Test Fail Notification",
			Steps: []mdk.Step{{ID: "send", Uses: "notification.send"}},
		}
		_ = runner.Register(wf)

		input := map[string]any{
			"recipient": recipient,
			"channel":   "EMAIL",
			"subject":   "Test",
			"body":      "Hello",
		}

		_, err := runner.ExecuteSync(context.Background(), "n2", "test-fail-wf", input)
		if err == nil {
			t.Fatal("expected workflow failure")
		}

		// DB should still have it as FAILED
		list, _ := mod.Repo().List(context.Background(), recipient)
		if len(list) != 1 || list[0].Status != StatusFailed {
			t.Error("expected 1 FAILED notification in repo")
		}
		
		mockProv.ShouldFail = false // Reset
	})
	
	t.Run("Event Subscriptions", func(t *testing.T) {
		recipient := fmt.Sprintf("event_%s@example.com", uuid.New().String()[:8])
		// Test identity.user_created
		rt.Bus().Publish(context.Background(), mdk.Event{
			Namespace: "identity",
			Type:      "user_created",
			Payload: map[string]any{
				"email": recipient,
				"name":  "Event User",
			},
		})
		
		// Wait for welcome email workflow to be executed asynchronously
		time.Sleep(100 * time.Millisecond)
		
		list, _ := mod.Repo().List(context.Background(), recipient)
		if len(list) != 1 {
			t.Error("expected welcome email to be sent")
		}
		
		// Test workflow.completed (fulfillment)
		rt.Bus().Publish(context.Background(), mdk.Event{
			Namespace: "workflow",
			Type:      "completed",
			Payload: map[string]any{
				"name": "fulfillment.v1",
			},
		})
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		// 1. Invalid input
		_, err := mod.SendNotification(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		
		// 2. Missing workflow input
		_, err = mod.SendNotification(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }
		
		// 3. Missing recipient
		_, err = mod.SendNotification(context.Background(), map[string]any{"input": map[string]any{}})
		if err == nil { t.Error("expected error for missing recipient") }
	})

	t.Run("Repository Edge Cases", func(t *testing.T) {
		repo := mod.Repo()
		ctx := context.Background()

		// 1. GetByID Not Found
		_, err := repo.GetByID(ctx, "ghost")
		if err == nil { t.Error("expected error for non-existent notif") }

		// 2. List with recipient filter
		n1 := &Notification{ID: "notif_1", Recipient: "user1", Status: StatusSent}
		n2 := &Notification{ID: "notif_2", Recipient: "user2", Status: StatusSent}
		repo.Save(ctx, n1)
		repo.Save(ctx, n2)

		list1, _ := repo.List(ctx, "user1")
		if len(list1) != 1 || list1[0].ID != "notif_1" { t.Error("List filter failed for user1") }

		listAll, _ := repo.List(ctx, "")
		if len(listAll) < 2 { t.Error("List with empty filter failed") }
	})
}
