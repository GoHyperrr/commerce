package store

import (
	"context"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for StoreSettings management.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

// NewModule creates a new instance of the Store module.
func NewModule() *Module {
	return &Module{}
}

// ID returns the unique identifier for the module.
func (m *Module) ID() string {
	return "commerce.store"
}

// Init initializes the store module and seeds default configurations if necessary.
func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Pre-seed default record if it doesn't exist in the database.
	var count int64
	if err := rt.DB().Model(&StoreSettings{}).Count(&count).Error; err == nil && count == 0 {
		defaultSettings := StoreSettings{
			ID:        1,
			Name:      "My Hyperrr Store",
			Host:      "localhost:8080",
			Email:     "admin@example.com",
			Currency:  "USD",
			Locale:    "en-US",
			Timezone:  "UTC",
			RobotsTXT: "User-agent: *\nDisallow:",
		}
		_ = rt.DB().Create(&defaultSettings)
	}

	return nil
}

// Shutdown cleans up resources on application stop.
func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

// Models registers GORM database structures for GORM auto-migration.
func (m *Module) Models() []any {
	return []any{&StoreSettings{}}
}

// Routes returns HTTP endpoints mapped to this module (none).
func (m *Module) Routes() []mdk.Route {
	return nil
}

// Repo returns the direct repository data-access client.
func (m *Module) Repo() *Repository {
	return m.repo
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
