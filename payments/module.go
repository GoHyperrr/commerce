package payments

import (
	"context"
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

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("payments.create_intent", m.handlers.CreateIntentStep)
	_ = rt.Workflows().RegisterHandler("payments.verify", m.handlers.VerifyPaymentStep)
	_ = rt.Workflows().RegisterHandler("payments.compensate", m.handlers.RefundStep)

	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
