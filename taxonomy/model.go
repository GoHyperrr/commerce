package taxonomy

import (
	"time"

	"github.com/GoHyperrr/commerce/seo"
	"github.com/GoHyperrr/mdk"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Taxonomy represents a decoupled categorization system (e.g., categories, tags, collections, brands).
type Taxonomy struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null" json:"name"`
	Code      string         `gorm:"uniqueIndex;not null" json:"code"`
	Type      string         `json:"type"` // e.g. "category", "tag", "collection", "brand"
	Metadata  mdk.Metadata   `gorm:"type:text" json:"metadata"`
	Terms     []TaxonomyTerm `gorm:"foreignKey:TaxonomyID;constraint:OnDelete:CASCADE" json:"terms"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Taxonomy) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// TaxonomyTerm represents an individual hierarchical term inside a Taxonomy.
type TaxonomyTerm struct {
	ID          string         `gorm:"primaryKey" json:"id"`
	TaxonomyID  string         `gorm:"index;not null" json:"taxonomy_id"`
	ParentID    *string        `gorm:"index" json:"parent_id"`
	Name        string         `gorm:"not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Description string         `json:"description"`
	SEO         seo.SEO        `gorm:"embedded;embeddedPrefix:seo_"`
	Metadata    mdk.Metadata   `gorm:"type:text" json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Children []TaxonomyTerm `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (tt *TaxonomyTerm) BeforeCreate(tx *gorm.DB) error {
	if tt.ID == "" {
		tt.ID = uuid.New().String()
	}
	return nil
}

// TaxonomyRelation creates a polymorphic link between a term and any other entity (e.g. product).
type TaxonomyRelation struct {
	TermID       string `gorm:"primaryKey" json:"term_id"`
	ResourceID   string `gorm:"primaryKey" json:"resource_id"`
	ResourceType string `gorm:"primaryKey" json:"resource_type"` // e.g. "product", "page", "post"
}

type CreateTaxonomyInput struct {
	Name     string       `json:"name"`
	Code     string       `json:"code"`
	Type     string       `json:"type"`
	Metadata mdk.Metadata `json:"metadata"`
}

type CreateTaxonomyTermInput struct {
	TaxonomyID  string       `json:"taxonomyId"`
	ParentID    *string      `json:"parentId"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description *string      `json:"description"`
	SEO         *SEOInput    `json:"seo"`
	Metadata    mdk.Metadata `json:"metadata"`
}

type SEOInput struct {
	MetaTitle       *string `json:"metaTitle"`
	MetaDescription *string `json:"metaDescription"`
	MetaKeywords    *string `json:"metaKeywords"`
	MetaImage       *string `json:"metaImage"`
}

type LinkResourceInput struct {
	TermID       string `json:"termId"`
	ResourceID   string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
}

type UnlinkResourceInput struct {
	TermID       string `json:"termId"`
	ResourceID   string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
}

