package payments

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/GoHyperrr/mdk"
)

type Module struct {
	rt       mdk.Runtime
	handlers *Handlers
	webhooks *WebhookHandlers
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.payments"
}

func (m *Module) Models() []any {
	return []any{&PaymentTransaction{}}
}

func (m *Module) Routes() []mdk.Route {
	return []mdk.Route{
		{
			Method:  "POST",
			Pattern: "/payments/stripe/webhook",
			Handler: http.HandlerFunc(m.webhooks.HandleStripe),
		},
		{
			Method:  "POST",
			Pattern: "/payments/razorpay/webhook",
			Handler: http.HandlerFunc(m.webhooks.HandleRazorpay),
		},
	}
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.handlers = NewHandlers(rt.DB(), rt)
	m.webhooks = NewWebhookHandlers(rt.DB(), rt)

	// Configure payment providers
	stripeKey, _ := rt.Config("stripe_secret_key").(string)
	if stripeKey == "" {
		stripeKey = os.Getenv("STRIPE_SECRET_KEY")
	}
	if stripeKey != "" {
		RegisterProvider(NewStripeProvider(stripeKey))
		rt.Logger().Info("Stripe payment provider enabled")
	}

	razorpayKeyID, _ := rt.Config("razorpay_key_id").(string)
	if razorpayKeyID == "" {
		razorpayKeyID = os.Getenv("RAZORPAY_KEY_ID")
	}
	razorpaySecret, _ := rt.Config("razorpay_key_secret").(string)
	if razorpaySecret == "" {
		razorpaySecret = os.Getenv("RAZORPAY_KEY_SECRET")
	}
	if razorpayKeyID != "" && razorpaySecret != "" {
		RegisterProvider(NewRazorpayProvider(razorpayKeyID, razorpaySecret))
		rt.Logger().Info("Razorpay payment provider enabled")
	}

	// Register workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "payments.create_intent",
		Name:        "payments.create_intent",
		Description: "Create a payment intent or order with Stripe, Razorpay, or Mock provider.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order_id": map[string]any{"type": "string"},
				"provider": map[string]any{"type": "string", "enum": []string{"stripe", "razorpay", "mock"}},
			},
			"required": []string{"order_id", "provider"},
		},
		Steps: []mdk.Step{
			{ID: "payments.create_intent", Name: "Create Intent", Uses: "payments.create_intent"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "payments.verify",
		Name:        "payments.verify",
		Description: "Verify a client-side payment signature or verification payload (e.g. Razorpay signature verification).",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order_id": map[string]any{"type": "string"},
				"provider": map[string]any{"type": "string", "enum": []string{"stripe", "razorpay", "mock"}},
				"payload":  map[string]any{"type": "object"},
			},
			"required": []string{"order_id", "provider", "payload"},
		},
		Steps: []mdk.Step{
			{ID: "payments.verify", Name: "Verify Signature", Uses: "payments.verify"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "payments.refund",
		Name:        "payments.refund",
		Description: "Process a payment refund/reversal.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"provider_transaction_id": map[string]any{"type": "string"},
				"amount":                  map[string]any{"type": "number"},
			},
			"required": []string{"provider_transaction_id", "amount"},
		},
		Steps: []mdk.Step{
			{ID: "refund", Name: "Refund", Uses: "payments.compensate"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "payments.get_transaction",
		Name:        "payments.get_transaction",
		Description: "Retrieve payment transaction details by transaction ID.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
			"required": []string{"id"},
		},
		Steps: []mdk.Step{
			{ID: "get", Name: "Get Transaction", Uses: "payments.get_transaction_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "payments.list_transactions",
		Name:        "payments.list_transactions",
		Description: "Retrieve a list of all payment transactions, optionally filtered by order ID.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order_id": map[string]any{"type": "string"},
			},
		},
		Steps: []mdk.Step{
			{ID: "list", Name: "List Transactions", Uses: "payments.list_transactions_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "payments.list_providers",
		Name:        "payments.list_providers",
		Description: "Retrieve a list of all active/configured payment providers.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
		},
		Steps: []mdk.Step{
			{ID: "list_providers", Name: "List Providers", Uses: "payments.list_providers_step"},
		},
	})

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("payments.create_intent", m.handlers.CreateIntentStep)
	_ = rt.Workflows().RegisterHandler("payments.verify", m.handlers.VerifyPaymentStep)
	_ = rt.Workflows().RegisterHandler("payments.compensate", m.handlers.RefundStep)
	_ = rt.Workflows().RegisterHandler("payments.get_transaction_step", m.GetTransactionStep)
	_ = rt.Workflows().RegisterHandler("payments.list_transactions_step", m.ListTransactionsStep)
	_ = rt.Workflows().RegisterHandler("payments.list_providers_step", m.ListProvidersStep)

	return nil
}

func (m *Module) GetTransactionStep(sCtx mdk.StepContext) mdk.StepResult {
	id, _ := sCtx.Input["id"].(string)
	if id == "" {
		return mdk.StepResult{Err: fmt.Errorf("id required")}
	}
	tx, err := m.GetPaymentTransaction(sCtx.Ctx, id)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"transaction": tx}}
}

func (m *Module) ListTransactionsStep(sCtx mdk.StepContext) mdk.StepResult {
	orderID, _ := sCtx.Input["order_id"].(string)
	if orderID != "" {
		txs, err := m.ListOrderTransactions(sCtx.Ctx, orderID)
		if err != nil {
			return mdk.StepResult{Err: err}
		}
		return mdk.StepResult{Output: map[string]any{"transactions": txs}}
	}

	var txs []*PaymentTransaction
	err := m.rt.DB().WithContext(sCtx.Ctx).Find(&txs).Error
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"transactions": txs}}
}

func (m *Module) ListProvidersStep(sCtx mdk.StepContext) mdk.StepResult {
	providers, err := m.ListActivePaymentProviders(sCtx.Ctx)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"providers": providers}}
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
