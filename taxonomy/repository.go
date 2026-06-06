package taxonomy

import (
	"context"

	"gorm.io/gorm"
)

// Repository handles database queries for Taxonomy, TaxonomyTerm, and TaxonomyRelation.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new taxonomy repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateTaxonomy inserts a new taxonomy record.
func (r *Repository) CreateTaxonomy(ctx context.Context, t *Taxonomy) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetTaxonomyByCode retrieves a taxonomy by its unique code.
func (r *Repository) GetTaxonomyByCode(ctx context.Context, code string) (*Taxonomy, error) {
	var t Taxonomy
	err := r.db.WithContext(ctx).Preload("Terms").Where("code = ?", code).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTaxonomies returns all taxonomies in the database.
func (r *Repository) ListTaxonomies(ctx context.Context) ([]Taxonomy, error) {
	var list []Taxonomy
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// CreateTerm inserts a new taxonomy term.
func (r *Repository) CreateTerm(ctx context.Context, term *TaxonomyTerm) error {
	return r.db.WithContext(ctx).Create(term).Error
}

// GetTermBySlug retrieves a taxonomy term by its slug.
func (r *Repository) GetTermBySlug(ctx context.Context, slug string) (*TaxonomyTerm, error) {
	var term TaxonomyTerm
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&term).Error
	if err != nil {
		return nil, err
	}
	return &term, nil
}

// GetTermTree builds and retrieves the nested terms hierarchy for a taxonomy.
func (r *Repository) GetTermTree(ctx context.Context, taxonomyID string) ([]TaxonomyTerm, error) {
	var terms []TaxonomyTerm
	err := r.db.WithContext(ctx).Where("taxonomy_id = ?", taxonomyID).Find(&terms).Error
	if err != nil {
		return nil, err
	}

	termMap := make(map[string]*TaxonomyTerm)
	var roots []TaxonomyTerm

	for i := range terms {
		termMap[terms[i].ID] = &terms[i]
	}

	for i := range terms {
		t := &terms[i]
		if t.ParentID == nil || *t.ParentID == "" {
			roots = append(roots, *t)
		} else {
			if parent, exists := termMap[*t.ParentID]; exists {
				parent.Children = append(parent.Children, *t)
			}
		}
	}

	// Re-map children updates to root elements
	for i := range roots {
		if updated, exists := termMap[roots[i].ID]; exists {
			roots[i] = *updated
		}
	}

	return roots, nil
}

// LinkResource associates a resource with a taxonomy term.
func (r *Repository) LinkResource(ctx context.Context, termID, resourceID, resourceType string) error {
	relation := TaxonomyRelation{
		TermID:       termID,
		ResourceID:   resourceID,
		ResourceType: resourceType,
	}
	return r.db.WithContext(ctx).FirstOrCreate(&relation).Error
}

// UnlinkResource removes the association between a resource and a taxonomy term.
func (r *Repository) UnlinkResource(ctx context.Context, termID, resourceID, resourceType string) error {
	return r.db.WithContext(ctx).
		Where("term_id = ? AND resource_id = ? AND resource_type = ?", termID, resourceID, resourceType).
		Delete(&TaxonomyRelation{}).Error
}

// GetTermsForResource retrieves all terms linked to a resource.
func (r *Repository) GetTermsForResource(ctx context.Context, resourceID, resourceType string) ([]TaxonomyTerm, error) {
	var relations []TaxonomyRelation
	err := r.db.WithContext(ctx).
		Where("resource_id = ? AND resource_type = ?", resourceID, resourceType).
		Find(&relations).Error
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 {
		return nil, nil
	}

	var termIDs []string
	for _, rel := range relations {
		termIDs = append(termIDs, rel.TermID)
	}

	var terms []TaxonomyTerm
	err = r.db.WithContext(ctx).Where("id IN ?", termIDs).Find(&terms).Error
	return terms, err
}

// GetResourceIDsForTerm retrieves all resource IDs of a specific type linked to a term.
func (r *Repository) GetResourceIDsForTerm(ctx context.Context, termID, resourceType string) ([]string, error) {
	var relations []TaxonomyRelation
	err := r.db.WithContext(ctx).
		Where("term_id = ? AND resource_type = ?", termID, resourceType).
		Find(&relations).Error
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, rel := range relations {
		ids = append(ids, rel.ResourceID)
	}
	return ids, nil
}
