package analytics

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Analytics.
type Module struct {
	rt mdk.Runtime
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) ID() string {
	return "commerce.analytics"
}

func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	return nil
}

func (m *Module) Models() []any {
	return nil
}

func (m *Module) Routes() []mdk.Route {
	return nil
}

func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

func (m *Module) Repo() any {
	return nil
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
