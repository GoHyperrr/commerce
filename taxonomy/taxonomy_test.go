package taxonomy

import (
	"context"
	"testing"

	"github.com/GoHyperrr/mdk/mdktest"
)

func TestTaxonomyModule(t *testing.T) {
	rt, _ := mdktest.NewInMemoryTestRuntime()

	mod := NewModule()
	_ = rt.DB().AutoMigrate(mod.Models()...)

	if err := mod.Init(context.Background(), rt); err != nil {
		t.Fatalf("failed to initialize taxonomy module: %v", err)
	}

	var taxID string
	var term1ID string
	var term2ID string

	t.Run("Create Taxonomy", func(t *testing.T) {
		input := CreateTaxonomyInput{
			Name: "Product Category",
			Code: "product_category",
			Type: "category",
		}
		tax, err := mod.CreateTaxonomy(context.Background(), input)
		if err != nil {
			t.Fatalf("failed to create taxonomy: %v", err)
		}
		if tax.ID == "" || tax.Code != "product_category" {
			t.Errorf("unexpected taxonomy created: %+v", tax)
		}
		taxID = tax.ID
	})

	t.Run("Create Taxonomy Terms", func(t *testing.T) {
		desc := "Electronics category"
		metaTitle := "Electronics SEO Title"
		inputRoot := CreateTaxonomyTermInput{
			TaxonomyID:  taxID,
			Name:        "Electronics",
			Slug:        "electronics",
			Description: &desc,
			SEO: &SEOInput{
				MetaTitle: &metaTitle,
			},
		}

		term1, err := mod.CreateTaxonomyTerm(context.Background(), inputRoot)
		if err != nil {
			t.Fatalf("failed to create root term: %v", err)
		}
		if term1.SEO.MetaTitle != "Electronics SEO Title" || term1.Slug != "electronics" {
			t.Errorf("unexpected term: %+v", term1)
		}
		term1ID = term1.ID

		descSub := "Laptops subcategory"
		inputSub := CreateTaxonomyTermInput{
			TaxonomyID:  taxID,
			ParentID:    &term1ID,
			Name:        "Laptops",
			Slug:        "laptops",
			Description: &descSub,
		}

		term2, err := mod.CreateTaxonomyTerm(context.Background(), inputSub)
		if err != nil {
			t.Fatalf("failed to create sub term: %v", err)
		}
		if term2.ParentID == nil || *term2.ParentID != term1ID {
			t.Errorf("unexpected sub term: %+v", term2)
		}
		term2ID = term2.ID
	})

	t.Run("Query Taxonomy and Term Tree", func(t *testing.T) {
		tax, err := mod.GetTaxonomy(context.Background(), "product_category")
		if err != nil {
			t.Fatalf("failed to get taxonomy: %v", err)
		}
		if len(tax.Terms) != 2 {
			t.Errorf("expected 2 terms, got %d", len(tax.Terms))
		}

		roots, err := mod.GetTermTree(context.Background(), taxID)
		if err != nil {
			t.Fatalf("failed to get term tree: %v", err)
		}
		if len(roots) != 1 {
			t.Errorf("expected 1 root term in tree, got %d", len(roots))
		}
		if len(roots[0].Children) != 1 || roots[0].Children[0].ID != term2ID {
			t.Errorf("unexpected term hierarchy in tree: %+v", roots[0])
		}
	})

	t.Run("Link and Unlink Resource", func(t *testing.T) {
		resID := "product_abc"
		resType := "product"

		// 1. Link
		linked, err := mod.LinkResource(context.Background(), LinkResourceInput{
			TermID:       term2ID,
			ResourceID:   resID,
			ResourceType: resType,
		})
		if err != nil || !linked {
			t.Fatalf("failed to link resource: %v", err)
		}

		// 2. Fetch associated terms
		terms, err := mod.GetTermsForResource(context.Background(), resID, resType)
		if err != nil || len(terms) != 1 || terms[0].ID != term2ID {
			t.Errorf("expected term %s linked to resource, got: %+v (err: %v)", term2ID, terms, err)
		}

		// 3. Fetch resource IDs for term
		ids, err := mod.GetResourceIdsForTerm(context.Background(), term2ID, resType)
		if err != nil || len(ids) != 1 || ids[0] != resID {
			t.Errorf("expected resource %s linked to term, got: %+v (err: %v)", resID, ids, err)
		}

		// 4. Unlink
		unlinked, err := mod.UnlinkResource(context.Background(), UnlinkResourceInput{
			TermID:       term2ID,
			ResourceID:   resID,
			ResourceType: resType,
		})
		if err != nil || !unlinked {
			t.Fatalf("failed to unlink resource: %v", err)
		}

		// 5. Verify unlinked
		terms, err = mod.GetTermsForResource(context.Background(), resID, resType)
		if err != nil || len(terms) != 0 {
			t.Errorf("expected 0 terms linked after unlink, got: %+v", terms)
		}
	})
}
