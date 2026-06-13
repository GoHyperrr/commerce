package taxonomy

import (
	"context"
)

// Queries exposes resolver query maps.
func (m *Module) Queries() map[string]any {
	return map[string]any{
		"taxonomy":           m.GetTaxonomy,
		"listTaxonomies":     m.ListTaxonomies,
		"term":               m.GetTerm,
		"termTree":           m.GetTermTree,
		"termsForResource":   m.GetTermsForResource,
		"resourceIdsForTerm": m.GetResourceIdsForTerm,
	}
}

// Mutations exposes resolver mutation maps.
func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"createTaxonomy":     m.CreateTaxonomy,
		"createTaxonomyTerm": m.CreateTaxonomyTerm,
		"linkResource":       m.LinkResource,
		"unlinkResource":     m.UnlinkResource,
	}
}

// FieldResolvers exposes custom field-level resolvers.
func (m *Module) FieldResolvers() map[string]any {
	return nil
}

// GetTaxonomy fetches a taxonomy by its unique code.
func (m *Module) GetTaxonomy(ctx context.Context, code string) (*Taxonomy, error) {
	return m.repo.GetTaxonomyByCode(ctx, code)
}

// ListTaxonomies lists all available taxonomies.
func (m *Module) ListTaxonomies(ctx context.Context) ([]*Taxonomy, error) {
	list, err := m.repo.ListTaxonomies(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*Taxonomy, len(list))
	for i := range list {
		res[i] = &list[i]
	}
	return res, nil
}

// GetTerm fetches an individual term by its unique slug.
func (m *Module) GetTerm(ctx context.Context, slug string) (*TaxonomyTerm, error) {
	return m.repo.GetTermBySlug(ctx, slug)
}

// GetTermTree retrieves the hierarchical term tree structure under a taxonomy.
func (m *Module) GetTermTree(ctx context.Context, taxonomyID string) ([]*TaxonomyTerm, error) {
	roots, err := m.repo.GetTermTree(ctx, taxonomyID)
	if err != nil {
		return nil, err
	}
	res := make([]*TaxonomyTerm, len(roots))
	for i := range roots {
		res[i] = &roots[i]
	}
	return res, nil
}

// GetTermsForResource retrieves all terms associated with a polymorphic resource.
func (m *Module) GetTermsForResource(ctx context.Context, resourceID string, resourceType string) ([]*TaxonomyTerm, error) {
	terms, err := m.repo.GetTermsForResource(ctx, resourceID, resourceType)
	if err != nil {
		return nil, err
	}
	res := make([]*TaxonomyTerm, len(terms))
	for i := range terms {
		res[i] = &terms[i]
	}
	return res, nil
}

// GetResourceIdsForTerm lists all associated resource IDs of a type linked to a term.
func (m *Module) GetResourceIdsForTerm(ctx context.Context, termID string, resourceType string) ([]string, error) {
	return m.repo.GetResourceIDsForTerm(ctx, termID, resourceType)
}

// CreateTaxonomy inserts a new taxonomy record.
func (m *Module) CreateTaxonomy(ctx context.Context, input CreateTaxonomyInput) (*Taxonomy, error) {
	t := &Taxonomy{
		Name:     input.Name,
		Code:     input.Code,
		Type:     input.Type,
		Metadata: input.Metadata,
	}
	err := m.repo.CreateTaxonomy(ctx, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// CreateTaxonomyTerm inserts a new taxonomy term.
func (m *Module) CreateTaxonomyTerm(ctx context.Context, input CreateTaxonomyTermInput) (*TaxonomyTerm, error) {
	var desc string
	if input.Description != nil {
		desc = *input.Description
	}

	term := &TaxonomyTerm{
		TaxonomyID:  input.TaxonomyID,
		ParentID:    input.ParentID,
		Name:        input.Name,
		Slug:        input.Slug,
		Description: desc,
		Metadata:    input.Metadata,
	}

	if input.SEO != nil {
		if input.SEO.MetaTitle != nil {
			term.SEO.MetaTitle = *input.SEO.MetaTitle
		}
		if input.SEO.MetaDescription != nil {
			term.SEO.MetaDescription = *input.SEO.MetaDescription
		}
		if input.SEO.MetaKeywords != nil {
			term.SEO.MetaKeywords = *input.SEO.MetaKeywords
		}
		if input.SEO.MetaImage != nil {
			term.SEO.MetaImage = *input.SEO.MetaImage
		}
	}

	err := m.repo.CreateTerm(ctx, term)
	if err != nil {
		return nil, err
	}
	return term, nil
}

// LinkResource connects a term to a resource.
func (m *Module) LinkResource(ctx context.Context, input LinkResourceInput) (bool, error) {
	err := m.repo.LinkResource(ctx, input.TermID, input.ResourceID, input.ResourceType)
	if err != nil {
		return false, err
	}
	return true, nil
}

// UnlinkResource removes the resource connection.
func (m *Module) UnlinkResource(ctx context.Context, input UnlinkResourceInput) (bool, error) {
	err := m.repo.UnlinkResource(ctx, input.TermID, input.ResourceID, input.ResourceType)
	if err != nil {
		return false, err
	}
	return true, nil
}
