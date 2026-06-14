package payments

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

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
		CustomerID string
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

	t.Run("AP2 Verification Success", func(t *testing.T) {
		// 1. Generate user and agent ECDSA key pairs
		userPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate user key: %v", err)
		}
		userPubBytes, err := x509.MarshalPKIXPublicKey(&userPriv.PublicKey)
		if err != nil {
			t.Fatalf("failed to marshal user public key: %v", err)
		}
		userPubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: userPubBytes}))

		agentPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate agent key: %v", err)
		}
		agentX := base64.RawURLEncoding.EncodeToString(agentPriv.X.Bytes())
		agentY := base64.RawURLEncoding.EncodeToString(agentPriv.Y.Bytes())

		// 2. Seed Customer record with the user public key in metadata
		err = db.Table("customers").AutoMigrate(&struct {
			ID       string `gorm:"primaryKey"`
			Metadata string
		}{})
		if err != nil {
			t.Fatalf("failed to migrate mock customers: %v", err)
		}

		metadataMap := map[string]any{"ap2_public_key": userPubPEM}
		metadataBytes, _ := json.Marshal(metadataMap)
		err = db.Table("customers").Create(map[string]any{
			"id":       "cust_ap2_test",
			"metadata": string(metadataBytes),
		}).Error
		if err != nil {
			t.Fatalf("failed to seed test customer: %v", err)
		}

		// 3. Seed Order record linked to customer
		err = db.Table("orders").Create(map[string]any{
			"id":          "order_ap2_test",
			"customer_id": "cust_ap2_test",
			"total_price": 50.00,
			"status":      "PENDING",
		}).Error
		if err != nil {
			t.Fatalf("failed to seed test order: %v", err)
		}

		// 4. Construct signed Mandate JWT
		mandateClaims := AP2MandateClaims{
			Issuer:     "did:example:user",
			Subject:    "did:example:agent",
			IssuedAt:   time.Now().Unix(),
			Expiration: time.Now().Add(1 * time.Hour).Unix(),
			CredentialSubject: AP2Subject{
				AgentKey: JWK{
					Kty: "EC",
					Crv: "P-256",
					X:   agentX,
					Y:   agentY,
				},
				Constraints: AP2Constraints{
					MaxAmount:         100.00,
					Currency:          "USD",
					MerchantAllowlist: []string{"hyperrr-store.com"},
				},
			},
		}

		mandateJWT, err := createSignedJWT(mandateClaims, userPriv)
		if err != nil {
			t.Fatalf("failed to sign mandate: %v", err)
		}

		// 5. Construct signed Agent Assertion JWT
		assertionClaims := AgentAssertionClaims{
			OrderID:  "order_ap2_test",
			Amount:   50.00,
			Currency: "USD",
			Nonce:    "test_nonce_123",
			IssuedAt: time.Now().Unix(),
		}

		agentAssertion, err := createSignedJWT(assertionClaims, agentPriv)
		if err != nil {
			t.Fatalf("failed to sign assertion: %v", err)
		}

		// 6. Verify and charge via handlers
		input := AP2VerificationInput{
			MandateJWT:     mandateJWT,
			AgentAssertion: agentAssertion,
			OrderID:        "order_ap2_test",
			Provider:       "mock",
		}

		tx, err := mod.handlers.VerifyAP2(ctx, input)
		if err != nil {
			t.Fatalf("failed to verify AP2 payment: %v", err)
		}

		if tx.Status != StatusSucceeded {
			t.Errorf("expected transaction status SUCCEEDED, got %s", tx.Status)
		}

		// Verify order was marked PAID
		var order struct {
			Status string
		}
		err = db.Table("orders").Where("id = ?", "order_ap2_test").First(&order).Error
		if err != nil {
			t.Fatalf("failed to fetch order: %v", err)
		}
		if order.Status != "PAID" {
			t.Errorf("expected order status PAID, got %s", order.Status)
		}
	})
}

func createSignedJWT(payload any, privKey *ecdsa.PrivateKey) (string, error) {
	header := map[string]string{"alg": "ES256", "typ": "JWT"}
	headerBytes, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)

	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := headerB64 + "." + payloadB64
	hash := sha256.Sum256([]byte(signingInput))

	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", err
	}

	rBytes := r.Bytes()
	sBytes := s.Bytes()
	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)

	sigBytes := append(rPadded, sPadded...)
	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return signingInput + "." + sigB64, nil
}
