package notification

import (
	"context"
	"fmt"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Notification.
type Module struct {
	repo     *Repository
	provider Provider
	rt       mdk.Runtime
}

func NewModule(provider Provider) *Module {
	if provider == nil {
		provider = &MockProvider{} // Default to mock
	}
	return &Module{provider: provider}
}

func (m *Module) ID() string {
	return "commerce.notification"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "notification.send_welcome",
		Name: "Send Welcome",
		Steps: []mdk.Step{
			{
				ID:   "send",
				Name: "Send Email",
				Uses: "notification.send",
			},
		},
	})

	// Register workflow step handlers
	_ = rt.Workflows().RegisterHandler("notification.send", m.SendNotificationStep)

	// Subscribe to Identity User Created
	_, _ = rt.Bus().Subscribe("identity", "user_created", func(ctx context.Context, event mdk.Event) error {
		email := getString(event.Payload, "email")
		name := getString(event.Payload, "name")

		input := map[string]any{
			"recipient": email,
			"channel":   string(ChannelEmail),
			"subject":   "Welcome to hyperrr!",
			"body":      fmt.Sprintf("Hi %s, thanks for joining.", name),
		}

		go rt.Workflows().Execute(ctx, "notification.send_welcome", input)
		return nil
	})

	// Subscribe to Order Completed (Workflow Completed)
	_, _ = rt.Bus().Subscribe("workflow", "completed", func(ctx context.Context, event mdk.Event) error {
		wfName := getString(event.Payload, "name")
		if wfName != "fulfillment.v1" {
			return nil
		}

		// In a real system, we'd fetch the order details here to get the email.
		// For this MVP, we'll just log that we would send it if we had the context easily available.
		rt.Logger().Info("Fulfillment completed, would send order confirmation email")

		return nil
	})

	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Models() []any {
	return []any{&Notification{}}
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) Repo() *Repository {
	return m.repo
}

// SendNotificationStep wraps SendNotification to mdk.StepHandler.
func (m *Module) SendNotificationStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.SendNotification(sCtx.Ctx, map[string]any{
		"input": sCtx.Input,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, ok := res.(*Notification)
	if ok {
		return mdk.StepResult{Output: map[string]any{"notification": resMap}}
	}
	return mdk.StepResult{}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule(nil)
	})
}
