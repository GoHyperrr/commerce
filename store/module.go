package store

import (
	"context"
	"encoding/json"
	"fmt"

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

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "store.get_settings",
		Name:        "store.get_settings",
		Description: "Retrieve the current global store settings.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
		},
		Steps: []mdk.Step{
			{ID: "get", Name: "Get Settings", Uses: "store.get_settings_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "store.update_settings",
		Name:        "store.update_settings",
		Description: "Update global store settings properties.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":            map[string]any{"type": "string"},
				"host":            map[string]any{"type": "string"},
				"email":           map[string]any{"type": "string"},
				"phone":           map[string]any{"type": "string"},
				"description":     map[string]any{"type": "string"},
				"currency":        map[string]any{"type": "string"},
				"locale":          map[string]any{"type": "string"},
				"timezone":        map[string]any{"type": "string"},
				"address":         map[string]any{"type": "string"},
				"city":            map[string]any{"type": "string"},
				"state":           map[string]any{"type": "string"},
				"zip":             map[string]any{"type": "string"},
				"country":         map[string]any{"type": "string"},
				"robotsTxt":       map[string]any{"type": "string"},
				"sitemapUrl":      map[string]any{"type": "string"},
				"logoUrl":         map[string]any{"type": "string"},
				"faviconUrl":      map[string]any{"type": "string"},
				"socialFacebook":  map[string]any{"type": "string"},
				"socialInstagram": map[string]any{"type": "string"},
				"socialTwitter":   map[string]any{"type": "string"},
				"socialLinkedin":  map[string]any{"type": "string"},
				"socialYoutube":   map[string]any{"type": "string"},
				"socialPinterest": map[string]any{"type": "string"},
				"socialTikTok":    map[string]any{"type": "string"},
			},
		},
		Steps: []mdk.Step{
			{ID: "update", Name: "Update Settings", Uses: "store.update_settings_step"},
		},
	})

	// Register step handlers
	_ = rt.Workflows().RegisterHandler("store.get_settings_step", m.GetSettingsStep)
	_ = rt.Workflows().RegisterHandler("store.update_settings_step", m.UpdateSettingsStep)

	return nil
}

func (m *Module) GetSettingsStep(sCtx mdk.StepContext) mdk.StepResult {
	settings, err := m.GetStoreSettings(sCtx.Ctx)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"settings": settings}}
}

func (m *Module) UpdateSettingsStep(sCtx mdk.StepContext) mdk.StepResult {
	ba, err := json.Marshal(sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	var in UpdateStoreSettingsInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return mdk.StepResult{Err: err}
	}

	settings, err := m.UpdateStoreSettings(sCtx.Ctx, in)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"settings": settings}}
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

func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	return []mdk.MCPResource{
		{
			URI:         "storesettings://current",
			Name:        "Store Settings",
			Description: "Current locale, currency, timezone, and brand info settings for the ecommerce store.",
			MimeType:    "application/json",
		},
	}, nil
}

func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	if uri != "storesettings://current" {
		return "", fmt.Errorf("resource not found")
	}
	settings, err := m.repo.Get(ctx)
	if err != nil {
		return "", err
	}
	dataBytes, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	return string(dataBytes), nil
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
