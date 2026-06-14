package payments

import (
	"context"
	"testing"

	"github.com/GoHyperrr/mdk/mdktest"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPayments(t *testing.T) {
	ctx := context.Background()

	// 1. Setup in-memory GORM database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// 2. Setup mock test runtime
	rt := mdktest.NewTestRuntime(db)

	mod := NewModule()
	// Run migrations
	err = db.AutoMigrate(mod.Models()...)
	if err != nil {
		t.Fatalf("failed to migrate models: %v", err)
	}

	// Seed an order for test
	err = db.Table("orders").AutoMigrate(&struct {
		ID         string  `gorm:"primaryKey"`
		TotalPrice float64
		Status     string
	}{})
	if err != nil {
		t.Fatalf("failed to migrate mock orders: %v", err)
	}

	err = db.Table("orders").Create(map[string]any{
		"id":          "order_123",
		"total_price": 49.99,
		"status":      "PENDING",
	}).Error
	if err != nil {
		t.Fatalf("failed to seed test order: %v", err)
	}

	// 3. Initialize Payments module
	err = mod.Init(ctx, rt)
	if err != nil {
		t.Fatalf("failed to initialize module: %v", err)
	}

	t.Run("Create Intent Mock Success", func(t *testing.T) {
		tx, err := mod.handlers.CreateIntent(ctx, "order_123", "mock")
		if err != nil {
			t.Fatalf("failed to create intent: %v", err)
		}

		if tx.OrderID != "order_123" {
			t.Errorf("expected order_123, got %s", tx.OrderID)
		}
		if tx.Amount != 49.99 {
			t.Errorf("expected 49.99, got %f", tx.Amount)
		}
		if tx.Status != StatusPending {
			t.Errorf("expected status PENDING, got %s", tx.Status)
		}
	})

	t.Run("Verify Signature Mock Success", func(t *testing.T) {
		payload := map[string]any{
			"razorpay_order_id":   "mock_order_123",
			"razorpay_payment_id": "mock_pay_123",
			"razorpay_signature":  "mock_sig_123",
		}

		tx, err := mod.handlers.VerifyPayment(ctx, "order_123", "mock", payload)
		if err != nil {
			t.Fatalf("failed to verify payment: %v", err)
		}

		if tx.Status != StatusSucceeded {
			t.Errorf("expected status SUCCEEDED, got %s", tx.Status)
		}

		// Verify that order status was updated to PAID
		var order struct {
			Status string
		}
		err = db.Table("orders").Where("id = ?", "order_123").First(&order).Error
		if err != nil {
			t.Fatalf("failed to fetch order: %v", err)
		}
		if order.Status != "PAID" {
			t.Errorf("expected order status PAID, got %s", order.Status)
		}
	})

	t.Run("Verify Signature Failure", func(t *testing.T) {
		// Re-seed an order and intent for failure test
		err = db.Table("orders").Create(map[string]any{
			"id":          "order_fail",
			"total_price": 100.00,
			"status":      "PENDING",
		}).Error
		if err != nil {
			t.Fatalf("failed to seed test order: %v", err)
		}

		_, err = mod.handlers.CreateIntent(ctx, "order_fail", "mock")
		if err != nil {
			t.Fatalf("failed to create intent: %v", err)
		}

		payload := map[string]any{
			"success": false,
		}

		_, err = mod.handlers.VerifyPayment(ctx, "order_fail", "mock", payload)
		if err == nil {
			t.Errorf("expected payment verification failure")
		}

		// Verify transaction is marked FAILED
		var tx PaymentTransaction
		err = db.Where("order_id = ? AND provider = ?", "order_fail", "mock").First(&tx).Error
		if err != nil {
			t.Fatalf("failed to fetch transaction: %v", err)
		}
		if tx.Status != StatusFailed {
			t.Errorf("expected transaction status FAILED, got %s", tx.Status)
		}
	})

	t.Run("Refund mock transaction", func(t *testing.T) {
		// Re-seed an order and intent for refund
		err = db.Table("orders").Create(map[string]any{
			"id":          "order_refund",
			"total_price": 20.00,
			"status":      "PENDING",
		}).Error
		if err != nil {
			t.Fatalf("failed to seed test order: %v", err)
		}

		tx, err := mod.handlers.CreateIntent(ctx, "order_refund", "mock")
		if err != nil {
			t.Fatalf("failed to create intent: %v", err)
		}

		payload := map[string]any{
			"success": true,
		}
		_, err = mod.handlers.VerifyPayment(ctx, "order_refund", "mock", payload)
		if err != nil {
			t.Fatalf("failed to verify payment: %v", err)
		}

		// Refund the transaction using the provider transaction id
		refID, err := mod.handlers.Refund(ctx, tx.ProviderTransactionID, 20.00)
		if err != nil {
			t.Fatalf("failed to refund transaction: %v", err)
		}

		if refID == "" {
			t.Errorf("expected refund ID, got empty string")
		}

		// Verify transaction is marked REFUNDED
		var updatedTx PaymentTransaction
		err = db.Where("id = ?", tx.ID).First(&updatedTx).Error
		if err != nil {
			t.Fatalf("failed to fetch updated transaction: %v", err)
		}
		if updatedTx.Status != StatusRefunded {
			t.Errorf("expected transaction status REFUNDED, got %s", updatedTx.Status)
		}
	})
}
