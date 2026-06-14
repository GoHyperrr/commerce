package store

import (
	"context"
	"testing"

	"github.com/GoHyperrr/mdk/mdktest"
)

func TestStoreSettings(t *testing.T) {
	rt, _ := mdktest.NewInMemoryTestRuntime()

	mod := NewModule()
	// Migrate models before Init to make sure tables exist for seeding count checks.
	_ = rt.DB().AutoMigrate(mod.Models()...)

	if err := mod.Init(context.Background(), rt); err != nil {
		t.Fatalf("failed to init module: %v", err)
	}

	t.Run("Default Seeding On Init", func(t *testing.T) {
		settings, err := mod.Repo().Get(context.Background())
		if err != nil {
			t.Fatalf("failed to fetch settings: %v", err)
		}
		if settings.ID != 1 {
			t.Errorf("expected ID 1, got %d", settings.ID)
		}
		if settings.Name != "My Hyperrr Store" {
			t.Errorf("expected name 'My Hyperrr Store', got %s", settings.Name)
		}
		if settings.Currency != "USD" {
			t.Errorf("expected currency 'USD', got %s", settings.Currency)
		}
	})

	t.Run("Get Store Settings GraphQL Helper", func(t *testing.T) {
		settings, err := mod.GetStoreSettings(context.Background())
		if err != nil {
			t.Fatalf("failed to get store settings: %v", err)
		}
		if settings.Locale != "en-US" {
			t.Errorf("expected en-US, got %s", settings.Locale)
		}
	})

	t.Run("Update Store Settings GraphQL Mutation Helper", func(t *testing.T) {
		nameUpdate := "New Store Name"
		currencyUpdate := "EUR"
		tiktokUpdate := "https://tiktok.com/@my_store"
		metaTitleUpdate := "SEO Meta Title"

		input := UpdateStoreSettingsInput{
			Name:         &nameUpdate,
			Currency:     &currencyUpdate,
			SocialTikTok: &tiktokUpdate,
			SEO: &SEOInput{
				MetaTitle: &metaTitleUpdate,
			},
		}

		updated, err := mod.UpdateStoreSettings(context.Background(), input)
		if err != nil {
			t.Fatalf("failed to update settings: %v", err)
		}

		if updated.Name != "New Store Name" {
			t.Errorf("expected name to be updated to 'New Store Name', got %s", updated.Name)
		}
		if updated.Currency != "EUR" {
			t.Errorf("expected currency to be updated to 'EUR', got %s", updated.Currency)
		}
		if updated.SocialTikTok != "https://tiktok.com/@my_store" {
			t.Errorf("expected TikTok link to be updated, got %s", updated.SocialTikTok)
		}
		if updated.SEO.MetaTitle != "SEO Meta Title" {
			t.Errorf("expected MetaTitle to be updated, got %s", updated.SEO.MetaTitle)
		}

		// Verify change persisted in database.
		dbRecord, err := mod.Repo().Get(context.Background())
		if err != nil {
			t.Fatalf("failed to fetch from DB: %v", err)
		}
		if dbRecord.Name != "New Store Name" {
			t.Errorf("expected database name to be updated, got %s", dbRecord.Name)
		}
	})
}
