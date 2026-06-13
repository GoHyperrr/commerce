package store

import (
	"time"

	"github.com/GoHyperrr/commerce/seo"
	"github.com/GoHyperrr/mdk"
	"gorm.io/gorm"
)

// StoreSettings represents the global configurations and settings of the e-commerce shop.
type StoreSettings struct {
	ID              uint           `gorm:"primaryKey;default:1" json:"id"`
	Name            string         `gorm:"not null;default:'My Hyperrr Store'" json:"name"`
	Host            string         `gorm:"not null;default:'localhost:8080'" json:"host"`
	Email           string         `gorm:"default:'admin@example.com'" json:"email"`
	Phone           string         `json:"phone"`
	Description     string         `json:"description"`
	Currency        string         `gorm:"not null;default:'USD'" json:"currency"`
	Locale          string         `gorm:"not null;default:'en-US'" json:"locale"`
	Timezone        string         `gorm:"not null;default:'UTC'" json:"timezone"`

	// Physical Store Address (used by fulfillment, tax, invoice modules)
	Address         string         `json:"address"`
	City            string         `json:"city"`
	State           string         `json:"state"`
	Zip             string         `json:"zip"`
	Country         string         `json:"country"`
	
	// SEO & Metadata
	SEO             seo.SEO        `gorm:"embedded;embeddedPrefix:seo_" json:"seo"`
	RobotsTXT       string         `gorm:"type:text" json:"robotsTxt"`
	SitemapURL      string         `json:"sitemapUrl"`
	LogoURL         string         `json:"logoUrl"`
	FaviconURL      string         `json:"faviconUrl"`

	// Social Links
	SocialFacebook  string         `json:"socialFacebook"`
	SocialInstagram string         `json:"socialInstagram"`
	SocialTwitter   string         `json:"socialTwitter"`
	SocialLinkedin  string         `json:"socialLinkedin"`
	SocialYoutube   string         `json:"socialYoutube"`
	SocialPinterest string         `json:"socialPinterest"`
	SocialTikTok    string         `json:"socialTikTok"`

	// Dynamic Metadata (Dynamic Extensibility options)
	Metadata        mdk.Metadata   `gorm:"type:text" json:"metadata"`

	// Timestamps
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// UpdateStoreSettingsInput maps GraphQL inputs to updates on the model.
type UpdateStoreSettingsInput struct {
	Name            *string       `json:"name"`
	Host            *string       `json:"host"`
	Email           *string       `json:"email"`
	Phone           *string       `json:"phone"`
	Description     *string       `json:"description"`
	Currency        *string       `json:"currency"`
	Locale          *string       `json:"locale"`
	Timezone        *string       `json:"timezone"`
	Address         *string       `json:"address"`
	City            *string       `json:"city"`
	State           *string       `json:"state"`
	Zip             *string       `json:"zip"`
	Country         *string       `json:"country"`
	SEO             *SEOInput     `json:"seo"`
	RobotsTxt       *string       `json:"robotsTxt"`
	SitemapUrl      *string       `json:"sitemapUrl"`
	LogoUrl         *string       `json:"logoUrl"`
	FaviconUrl      *string       `json:"faviconUrl"`
	SocialFacebook  *string       `json:"socialFacebook"`
	SocialInstagram *string       `json:"socialInstagram"`
	SocialTwitter   *string       `json:"socialTwitter"`
	SocialLinkedin  *string       `json:"socialLinkedin"`
	SocialYoutube   *string       `json:"socialYoutube"`
	SocialPinterest *string       `json:"socialPinterest"`
	SocialTikTok    *string       `json:"socialTikTok"`
	Metadata        mdk.Metadata  `json:"metadata"`
}

// SEOInput matches the GraphQL input mapping.
type SEOInput struct {
	MetaTitle       *string `json:"metaTitle"`
	MetaDescription *string `json:"metaDescription"`
	MetaKeywords    *string `json:"metaKeywords"`
	MetaImage       *string `json:"metaImage"`
}
