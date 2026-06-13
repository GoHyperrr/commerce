package store

import (
	"context"

	"gorm.io/gorm"
)

// Queries exposes resolver query maps.
func (m *Module) Queries() map[string]any {
	return map[string]any{
		"storeSettings": m.GetStoreSettings,
	}
}

// Mutations exposes resolver mutation maps.
func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"updateStoreSettings": m.UpdateStoreSettings,
	}
}

// FieldResolvers exposes custom field-level resolvers.
func (m *Module) FieldResolvers() map[string]any {
	return nil
}

// GetStoreSettings returns the global store settings record.
func (m *Module) GetStoreSettings(ctx context.Context) (*StoreSettings, error) {
	settings, err := m.repo.Get(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Seed default settings on the fly if not exists
			defaultSettings := &StoreSettings{
				ID:        1,
				Name:      "My Hyperrr Store",
				Host:      "localhost:8080",
				Email:     "admin@example.com",
				Currency:  "USD",
				Locale:    "en-US",
				Timezone:  "UTC",
				RobotsTXT: "User-agent: *\nDisallow:",
			}
			if saveErr := m.repo.Save(ctx, defaultSettings); saveErr != nil {
				return nil, saveErr
			}
			return defaultSettings, nil
		}
		return nil, err
	}
	return settings, nil
}

// UpdateStoreSettings updates non-nil input configurations for the store settings.
func (m *Module) UpdateStoreSettings(ctx context.Context, input UpdateStoreSettingsInput) (*StoreSettings, error) {
	settings, err := m.GetStoreSettings(ctx)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		settings.Name = *input.Name
	}
	if input.Host != nil {
		settings.Host = *input.Host
	}
	if input.Email != nil {
		settings.Email = *input.Email
	}
	if input.Phone != nil {
		settings.Phone = *input.Phone
	}
	if input.Description != nil {
		settings.Description = *input.Description
	}
	if input.Currency != nil {
		settings.Currency = *input.Currency
	}
	if input.Locale != nil {
		settings.Locale = *input.Locale
	}
	if input.Timezone != nil {
		settings.Timezone = *input.Timezone
	}
	if input.Address != nil {
		settings.Address = *input.Address
	}
	if input.City != nil {
		settings.City = *input.City
	}
	if input.State != nil {
		settings.State = *input.State
	}
	if input.Zip != nil {
		settings.Zip = *input.Zip
	}
	if input.Country != nil {
		settings.Country = *input.Country
	}
	if input.RobotsTxt != nil {
		settings.RobotsTXT = *input.RobotsTxt
	}
	if input.SitemapUrl != nil {
		settings.SitemapURL = *input.SitemapUrl
	}
	if input.LogoUrl != nil {
		settings.LogoURL = *input.LogoUrl
	}
	if input.FaviconUrl != nil {
		settings.FaviconURL = *input.FaviconUrl
	}
	if input.SocialFacebook != nil {
		settings.SocialFacebook = *input.SocialFacebook
	}
	if input.SocialInstagram != nil {
		settings.SocialInstagram = *input.SocialInstagram
	}
	if input.SocialTwitter != nil {
		settings.SocialTwitter = *input.SocialTwitter
	}
	if input.SocialLinkedin != nil {
		settings.SocialLinkedin = *input.SocialLinkedin
	}
	if input.SocialYoutube != nil {
		settings.SocialYoutube = *input.SocialYoutube
	}
	if input.SocialPinterest != nil {
		settings.SocialPinterest = *input.SocialPinterest
	}
	if input.SocialTikTok != nil {
		settings.SocialTikTok = *input.SocialTikTok
	}
	if input.SEO != nil {
		if input.SEO.MetaTitle != nil {
			settings.SEO.MetaTitle = *input.SEO.MetaTitle
		}
		if input.SEO.MetaDescription != nil {
			settings.SEO.MetaDescription = *input.SEO.MetaDescription
		}
		if input.SEO.MetaKeywords != nil {
			settings.SEO.MetaKeywords = *input.SEO.MetaKeywords
		}
		if input.SEO.MetaImage != nil {
			settings.SEO.MetaImage = *input.SEO.MetaImage
		}
	}
	if input.Metadata != nil {
		settings.Metadata = input.Metadata
	}

	err = m.repo.Save(ctx, settings)
	if err != nil {
		return nil, err
	}

	return settings, nil
}
