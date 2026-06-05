package support

import (
	"context"
	"os"
	"testing"

	"github.com/GoHyperrr/mdk"
	"github.com/GoHyperrr/mdk/mdktest"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSupportModule(t *testing.T) {
	dbFile := "support_test.db"
	defer os.Remove(dbFile)

	database, _ := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	rt := mdktest.NewTestRuntime(database)

	mod := NewModule()
	_ = mod.Init(context.Background(), rt)
	_ = database.AutoMigrate(mod.Models()...)
	runner := rt.Workflows().(*mdktest.TestWorkflowEngine)

	t.Run("Create Ticket Success", func(t *testing.T) {
		wf := mdk.Workflow{
			ID:    "test-support-wf",
			Name:  "Test Support",
			Steps: []mdk.Step{{ID: "ticket", Uses: "support.create_ticket"}},
		}
		_ = runner.Register(wf)

		input := map[string]any{
			"customer_id": "cust1",
			"subject":     "Help!",
			"message":     "Need help with my order.",
		}

		res, err := runner.ExecuteSync(context.Background(), "t1", "test-support-wf", input)
		if err != nil {
			t.Fatalf("workflow failed: %v", err)
		}

		resMap := res["ticket"].(map[string]any)
		ticket := resMap["ticket"].(*Ticket)
		if ticket.Status != TicketOpen || len(ticket.Messages) != 1 {
			t.Error("ticket creation failed")
		}
	})

	t.Run("AI Dispatch Response", func(t *testing.T) {
		tkt := &Ticket{ID: "tkt2", CustomerID: "cust2", Status: TicketOpen}
		mod.Repo().SaveTicket(context.Background(), tkt)

		// Test normal response
		results := map[string]any{
			"ticket": map[string]any{"ticket": tkt},
			"input":  map[string]any{},
		}

		res, err := mod.DispatchAIResponse(context.Background(), results)
		if err != nil {
			t.Fatalf("DispatchAIResponse failed: %v", err)
		}

		resMap := res.(map[string]any)
		msg := resMap["message"].(*Message)
		if msg.Sender != SenderAI {
			t.Error("expected AI sender")
		}
	})

	t.Run("Handler Error Cases", func(t *testing.T) {
		// CreateTicket - Invalid Input
		_, err := mod.CreateTicket(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.CreateTicket(context.Background(), map[string]any{"wrong": 1})
		if err == nil { t.Error("expected error for missing workflow input") }

		// DispatchAIResponse - Invalid Input
		_, err = mod.DispatchAIResponse(context.Background(), "string")
		if err == nil { t.Error("expected error for invalid input type") }
		_, err = mod.DispatchAIResponse(context.Background(), map[string]any{})
		if err == nil { t.Error("expected error for missing result from ticket step") }
		_, err = mod.DispatchAIResponse(context.Background(), map[string]any{"ticket": "not-a-map"})
		if err == nil { t.Error("expected error for invalid result format") }
		_, err = mod.DispatchAIResponse(context.Background(), map[string]any{"ticket": map[string]any{"wrong": 1}})
		if err == nil { t.Error("expected error for missing ticket in map") }

		// DispatchAIResponse - No special setup needed for default
		tkt := &Ticket{ID: "tkt_no_proj", CustomerID: "cust1", Status: TicketOpen}
		res, err := mod.DispatchAIResponse(context.Background(), map[string]any{"ticket": map[string]any{"ticket": tkt}})
		if err != nil { t.Fatalf("failed without projector: %v", err) }
		if res.(map[string]any)["message"].(*Message).Content != "Hello! I am your AI assistant. How can I help you today?" {
			t.Error("expected default AI message")
		}

		// DispatchAIResponse - Database failure
		badMod := NewModule()
		dbFile := "support_bad.db"
		defer os.Remove(dbFile)
		badDB, _ := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
		sqlDB, _ := badDB.DB()
		badMod.repo = NewRepository(badDB)
		sqlDB.Close()
		_, err = badMod.DispatchAIResponse(context.Background(), map[string]any{"ticket": map[string]any{"ticket": tkt}})
		if err == nil { t.Error("expected error on DB failure") }
	})
}

func TestSupportRepository(t *testing.T) {
	dbFile := "support_repo_test.db"
	defer os.Remove(dbFile)
	database, _ := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	
	repo := NewRepository(database)
	_ = database.AutoMigrate(&Ticket{}, &Message{})

	t.Run("CRUD", func(t *testing.T) {
		tkt := &Ticket{ID: "t1", CustomerID: "c1", Status: TicketOpen, Subject: "S"}
		err := repo.SaveTicket(context.Background(), tkt)
		if err != nil { t.Error(err) }

		msg := &Message{ID: "m1", TicketID: "t1", Content: "C", Sender: SenderHuman}
		repo.SaveMessage(context.Background(), msg)

		got, _ := repo.GetTicketByID(context.Background(), "t1")
		if len(got.Messages) != 1 { t.Error("Preload messages failed") }

		list, _ := repo.ListTicketsByCustomerID(context.Background(), "c1")
		if len(list) != 1 { t.Error("ListTicketsByCustomerID failed") }
	})
}
