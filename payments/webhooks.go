package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"gorm.io/gorm"
)

type WebhookHandlers struct {
	db *gorm.DB
	rt mdk.Runtime
}

func NewWebhookHandlers(db *gorm.DB, rt mdk.Runtime) *WebhookHandlers {
	return &WebhookHandlers{
		db: db,
		rt: rt,
	}
}

func (wh *WebhookHandlers) HandleStripe(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	webhookSecret, _ := wh.rt.Config("stripe_webhook_secret").(string)

	var event stripe.Event

	// Verify webhook signature if secret is configured
	if webhookSecret != "" {
		event, err = webhook.ConstructEvent(payload, sigHeader, webhookSecret)
		if err != nil {
			wh.rt.Logger().Error("Stripe webhook signature verification failed", "error", err)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	} else {
		// Fallback for local testing when webhook secret is omitted
		if err := json.Unmarshal(payload, &event); err != nil {
			http.Error(w, "failed to parse payload", http.StatusBadRequest)
			return
		}
	}

	wh.rt.Logger().Info("Received Stripe Webhook", "type", event.Type, "id", event.ID)

	if event.Type == "payment_intent.succeeded" {
		var pi stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &pi)
		if err != nil {
			wh.rt.Logger().Error("failed to parse Stripe payment intent data", "error", err)
			http.Error(w, "parsing failed", http.StatusBadRequest)
			return
		}

		wh.processSucceededPayment(pi.ID, "stripe")
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *WebhookHandlers) HandleRazorpay(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("X-Razorpay-Signature")
	webhookSecret, _ := wh.rt.Config("razorpay_webhook_secret").(string)

	// Verify signature if webhook secret is configured
	if webhookSecret != "" {
		h := hmac.New(sha256.New, []byte(webhookSecret))
		_, _ = h.Write(payload)
		expectedSignature := hex.EncodeToString(h.Sum(nil))
		if !hmac.Equal([]byte(expectedSignature), []byte(sigHeader)) {
			wh.rt.Logger().Error("Razorpay webhook signature verification failed")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse event
	var event struct {
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity struct {
					OrderID string `json:"order_id"`
					ID      string `json:"id"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "failed to parse Razorpay payload", http.StatusBadRequest)
		return
	}

	wh.rt.Logger().Info("Received Razorpay Webhook", "event", event.Event)

	if event.Event == "payment.captured" {
		// Razorpay triggers capture and webhook. We match by Razorpay Order ID.
		wh.processSucceededPayment(event.Payload.Payment.Entity.OrderID, "razorpay")
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *WebhookHandlers) processSucceededPayment(providerTxID string, provider string) {
	var tx PaymentTransaction
	err := wh.db.Where("provider_transaction_id = ? AND provider = ? AND status = ?", providerTxID, provider, StatusPending).First(&tx).Error
	if err != nil {
		wh.rt.Logger().Warn("Transaction not found or already processed for webhook", "provider_tx_id", providerTxID)
		return
	}

	tx.Status = StatusSucceeded
	if err := wh.db.Save(&tx).Error; err != nil {
		wh.rt.Logger().Error("failed to update transaction state on webhook", "tx_id", tx.ID, "error", err)
		return
	}

	// Update order status to PAID in GORM
	err = wh.db.Table("orders").Where("id = ?", tx.OrderID).Update("status", "PAID").Error
	if err != nil {
		wh.rt.Logger().Error("failed to update order status to paid on webhook", "order_id", tx.OrderID, "error", err)
	}

	// Emit order.paid event
	_ = wh.rt.Bus().Publish(context.Background(), mdk.Event{
		ID:        "evt_" + uuid.New().String(),
		Namespace: "commerce.order",
		Type:      "paid",
		Payload: map[string]any{
			"order_id":       tx.OrderID,
			"amount":         tx.Amount,
			"provider":       provider,
			"transaction_id": tx.ID,
		},
		OccurredAt: time.Now(),
	})

	wh.rt.Logger().Info("Processed webhook successful payment", "order_id", tx.OrderID, "transaction_id", tx.ID)
}
