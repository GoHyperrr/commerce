package marketing

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Marketing.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.marketing"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:   "marketing.apply_coupon",
		Name: "Marketing Apply Coupon",
		Steps: []mdk.Step{
			{ID: "validate", Name: "Validate Coupon", Uses: "marketing.validate_coupon"},
			{ID: "apply", Name: "Apply Coupon", Uses: "cart.add_item", DependsOn: []string{"validate"}},
		},
	})

	// Register named workflow step handlers
	_ = rt.Workflows().RegisterHandler("marketing.validate_coupon", m.ValidateCouponStep)
	_ = rt.Workflows().RegisterHandler("marketing.add_loyalty_points", m.AddLoyaltyPointsStep)

	return nil
}

func (m *Module) Models() []any {
	return []any{&Coupon{}, &LoyaltyPoints{}}
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Repo() *Repository {
	return m.repo
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
