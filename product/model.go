package product

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/GoHyperrr/commerce/seo"
	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringList represents a slice of strings serialized as JSON text in the database.
type StringList []string

// Value returns the driver Value.
func (sl StringList) Value() (driver.Value, error) {
	if sl == nil {
		return "[]", nil
	}
	ba, err := json.Marshal(sl)
	if err != nil {
		return nil, err
	}
	return string(ba), nil
}

// Scan scans value into StringList.
func (sl *StringList) Scan(val interface{}) error {
	if val == nil {
		*sl = nil
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New("failed to scan StringList: invalid type")
	}
	return json.Unmarshal(ba, sl)
}

// ProductDetails groups package shipping specifications and digital download details.
type ProductDetails struct {
	SKU                      *string  `gorm:"column:sku" json:"sku,omitempty"`
	Barcode                  *string  `gorm:"column:barcode" json:"barcode,omitempty"`
	HSNCode                  *string  `gorm:"column:hsn_code" json:"hsn_code,omitempty"`
	Weight                   *float64 `gorm:"column:weight" json:"weight,omitempty"`
	Length                   *float64 `gorm:"column:length" json:"length,omitempty"`
	Width                    *float64 `gorm:"column:width" json:"width,omitempty"`
	Height                   *float64 `gorm:"column:height" json:"height,omitempty"`
	FileUrl                  *string  `gorm:"column:file_url" json:"file_url,omitempty"`
	MaxDownloads             *int     `gorm:"column:max_downloads" json:"max_downloads,omitempty"`
	DownloadExpirationHours *int     `gorm:"column:download_expiration_hours" json:"download_expiration_hours,omitempty"`
}

// Product represents the main catalog entity.
type Product struct {
	ID              string           `gorm:"primaryKey" json:"id"`
	Name            string           `gorm:"not null" json:"name"`
	Handle          string           `gorm:"uniqueIndex;not null" json:"handle"`
	Description     string           `json:"description"`
	Type            string           `gorm:"default:PHYSICAL" json:"type"` // PHYSICAL, DIGITAL, SERVICE
	Status          string           `gorm:"default:DRAFT" json:"status"`   // DRAFT, PUBLISHED, ARCHIVED
	Details         ProductDetails   `gorm:"embedded;embeddedPrefix:details_" json:"details"`
	SEO             seo.SEO          `gorm:"embedded;embeddedPrefix:seo_" json:"seo"`
	Metadata        mdk.Metadata     `gorm:"type:text" json:"metadata"`
	AISystemContext string           `json:"ai_system_context"`
	Options         []ProductOption  `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"options"`
	Variants        []ProductVariant `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"variants"`
	Images          []ProductImage   `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"images"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `gorm:"index" json:"-"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// ProductOption defines the option names (e.g., Color, Size) and allowed values for variations.
type ProductOption struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	ProductID string     `gorm:"index;not null" json:"product_id"`
	Name      string     `gorm:"not null" json:"name"`
	Values    StringList `gorm:"type:text" json:"values"`
}

func (po *ProductOption) BeforeCreate(tx *gorm.DB) error {
	if po.ID == "" {
		po.ID = uuid.New().String()
	}
	return nil
}

// ProductVariant represents a stockable configuration of a product with its own price.
type ProductVariant struct {
	ID             string          `gorm:"primaryKey" json:"id"`
	ProductID      string          `gorm:"index;not null" json:"product_id"`
	Title          string          `gorm:"not null" json:"title"`
	Price          float64         `gorm:"not null" json:"price"`
	CompareAtPrice *float64        `json:"compare_at_price,omitempty"`
	Details        ProductDetails  `gorm:"embedded;embeddedPrefix:details_" json:"details"`
	Options        []VariantOption `gorm:"foreignKey:VariantID;constraint:OnDelete:CASCADE" json:"options"`
	Metadata       mdk.Metadata    `gorm:"type:text" json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (pv *ProductVariant) BeforeCreate(tx *gorm.DB) error {
	if pv.ID == "" {
		pv.ID = uuid.New().String()
	}
	return nil
}

// VariantOption defines the key-value options chosen for a specific variant (e.g. Size: "XL").
type VariantOption struct {
	ID        string `gorm:"primaryKey" json:"id"`
	VariantID string `gorm:"index;not null" json:"variant_id"`
	Name      string `gorm:"not null" json:"name"`
	Value     string `gorm:"not null" json:"value"`
}

func (vo *VariantOption) BeforeCreate(tx *gorm.DB) error {
	if vo.ID == "" {
		vo.ID = uuid.New().String()
	}
	return nil
}

// ProductImage represents media links associated with the product or particular variants.
type ProductImage struct {
	ID        string  `gorm:"primaryKey" json:"id"`
	ProductID string  `gorm:"index;not null" json:"product_id"`
	VariantID *string `gorm:"index" json:"variant_id,omitempty"`
	URL       string  `gorm:"not null" json:"url"`
	AltText   string  `json:"alt_text"`
	SortOrder int     `gorm:"default:0" json:"sort_order"`
}

func (pi *ProductImage) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == "" {
		pi.ID = uuid.New().String()
	}
	return nil
}

// GraphQL input mapping structures. Defined locally in the commerce/product package
// to avoid importing hyperrr-specific structures, preventing circular compilation loops.

type CreateProductInput struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Handle          string                      `json:"handle"`
	Description     *string                     `json:"description,omitempty"`
	Type            *string                     `json:"type,omitempty"`
	Status          *string                     `json:"status,omitempty"`
	Details         *ProductDetailsInput        `json:"details,omitempty"`
	SEO             *SEOInput                   `json:"seo,omitempty"`
	Metadata        mdk.Metadata                `json:"metadata,omitempty"`
	AISystemContext *string                     `json:"ai_system_context,omitempty"`
	Options         []ProductOptionInput        `json:"options,omitempty"`
	Variants        []CreateProductVariantInput `json:"variants,omitempty"`
	Images          []ProductImageInput         `json:"images,omitempty"`
}

type ProductDetailsInput struct {
	SKU                      *string  `json:"sku,omitempty"`
	Barcode                  *string  `json:"barcode,omitempty"`
	HSNCode                  *string  `json:"hsn_code,omitempty"`
	Weight                   *float64 `json:"weight,omitempty"`
	Length                   *float64 `json:"length,omitempty"`
	Width                    *float64 `json:"width,omitempty"`
	Height                   *float64 `json:"height,omitempty"`
	FileUrl                  *string  `json:"file_url,omitempty"`
	MaxDownloads             *int     `json:"max_downloads,omitempty"`
	DownloadExpirationHours *int     `json:"download_expiration_hours,omitempty"`
}

type SEOInput struct {
	MetaTitle       *string `json:"metaTitle,omitempty"`
	MetaDescription *string `json:"metaDescription,omitempty"`
	MetaKeywords    *string `json:"metaKeywords,omitempty"`
	MetaImage       *string `json:"metaImage,omitempty"`
}

type ProductOptionInput struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type CreateProductVariantInput struct {
	Title          string               `json:"title"`
	Price          float64              `json:"price"`
	CompareAtPrice *float64             `json:"compare_at_price,omitempty"`
	Details        *ProductDetailsInput `json:"details,omitempty"`
	Options        []VariantOptionInput `json:"options,omitempty"`
	Metadata       mdk.Metadata         `json:"metadata,omitempty"`
}

type VariantOptionInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProductImageInput struct {
	VariantID *string `json:"variant_id,omitempty"`
	URL       string  `json:"url"`
	AltText   *string `json:"alt_text,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

type UpdateProductInput struct {
	Name            *string              `json:"name,omitempty"`
	Handle          *string              `json:"handle,omitempty"`
	Description     *string              `json:"description,omitempty"`
	Type            *string              `json:"type,omitempty"`
	Status          *string              `json:"status,omitempty"`
	Details         *ProductDetailsInput `json:"details,omitempty"`
	SEO             *SEOInput            `json:"seo,omitempty"`
	Metadata        mdk.Metadata         `json:"metadata,omitempty"`
	AISystemContext *string              `json:"ai_system_context,omitempty"`
}
