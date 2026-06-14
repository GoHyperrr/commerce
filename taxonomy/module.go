package taxonomy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/mdk"
)

// Module implements the mdk.Module interface for Taxonomy.
type Module struct {
	repo *Repository
	rt   mdk.Runtime
}

// NewModule creates a new instance of the taxonomy module.
func NewModule() *Module {
	return &Module{}
}

// ID returns the unique identifier for the taxonomy module.
func (m *Module) ID() string {
	return "commerce.taxonomy"
}

// Init initializes the repository and database connections.
func (m *Module) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	m.repo = NewRepository(rt.DB())

	// Register Workflows
	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.create",
		Name:        "taxonomy.create",
		Description: "Create a new taxonomy category system (e.g. tag, category, brand).",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"code": map[string]any{"type": "string"},
				"type": map[string]any{"type": "string"},
			},
			"required": []string{"name", "code", "type"},
		},
		Steps: []mdk.Step{
			{ID: "create", Name: "Create Taxonomy", Uses: "taxonomy.create_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.create_term",
		Name:        "taxonomy.create_term",
		Description: "Create a term (value) inside a taxonomy category system.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"taxonomyId":  map[string]any{"type": "string"},
				"name":        map[string]any{"type": "string"},
				"slug":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"parentId":    map[string]any{"type": "string"},
			},
			"required": []string{"taxonomyId", "name", "slug"},
		},
		Steps: []mdk.Step{
			{ID: "create_term", Name: "Create Term", Uses: "taxonomy.create_term_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.link_resource",
		Name:        "taxonomy.link_resource",
		Description: "Link a resource (like a product) to a taxonomy term (e.g. associate a product with a category).",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"termId":       map[string]any{"type": "string"},
				"resourceId":   map[string]any{"type": "string"},
				"resourceType": map[string]any{"type": "string"},
			},
			"required": []string{"termId", "resourceId", "resourceType"},
		},
		Steps: []mdk.Step{
			{ID: "link", Name: "Link Resource", Uses: "taxonomy.link_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.list",
		Name:        "taxonomy.list",
		Description: "Retrieve a list of all taxonomy category systems.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
		},
		Steps: []mdk.Step{
			{ID: "list", Name: "List Taxonomies", Uses: "taxonomy.list_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.get_term_tree",
		Name:        "taxonomy.get_term_tree",
		Description: "Retrieve the hierarchical term tree structure under a taxonomy.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"taxonomyId": map[string]any{"type": "string"},
			},
			"required": []string{"taxonomyId"},
		},
		Steps: []mdk.Step{
			{ID: "get_term_tree", Name: "Get Term Tree", Uses: "taxonomy.get_term_tree_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.get_by_code",
		Name:        "taxonomy.get_by_code",
		Description: "Retrieve details of a taxonomy by its unique code.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{"type": "string"},
			},
			"required": []string{"code"},
		},
		Steps: []mdk.Step{
			{ID: "get_by_code", Name: "Get Taxonomy By Code", Uses: "taxonomy.get_by_code_step"},
		},
	})

	_ = rt.Workflows().Register(mdk.Workflow{
		ID:          "taxonomy.get_term_by_slug",
		Name:        "taxonomy.get_term_by_slug",
		Description: "Retrieve details of an individual term by its unique slug.",
		ExposeToAI:  true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"slug": map[string]any{"type": "string"},
			},
			"required": []string{"slug"},
		},
		Steps: []mdk.Step{
			{ID: "get_term_by_slug", Name: "Get Term By Slug", Uses: "taxonomy.get_term_by_slug_step"},
		},
	})

	_ = rt.Workflows().RegisterHandler("taxonomy.create_step", m.CreateTaxonomyStep)
	_ = rt.Workflows().RegisterHandler("taxonomy.create_term_step", m.CreateTermStep)
	_ = rt.Workflows().RegisterHandler("taxonomy.link_step", m.LinkStep)
	_ = rt.Workflows().RegisterHandler("taxonomy.list_step", m.ListTaxonomiesStep)
	_ = rt.Workflows().RegisterHandler("taxonomy.get_term_tree_step", m.GetTermTreeStep)
	_ = rt.Workflows().RegisterHandler("taxonomy.get_by_code_step", m.GetTaxonomyByCodeStep)
	_ = rt.Workflows().RegisterHandler("taxonomy.get_term_by_slug_step", m.GetTermBySlugStep)

	return nil
}

func (m *Module) ListTaxonomiesStep(sCtx mdk.StepContext) mdk.StepResult {
	list, err := m.ListTaxonomies(sCtx.Ctx)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"taxonomies": list}}
}

func (m *Module) GetTermTreeStep(sCtx mdk.StepContext) mdk.StepResult {
	taxonomyID, _ := sCtx.Input["taxonomyId"].(string)
	if taxonomyID == "" {
		return mdk.StepResult{Err: fmt.Errorf("taxonomyId required")}
	}
	roots, err := m.GetTermTree(sCtx.Ctx, taxonomyID)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"terms": roots}}
}

func (m *Module) GetTaxonomyByCodeStep(sCtx mdk.StepContext) mdk.StepResult {
	code, _ := sCtx.Input["code"].(string)
	if code == "" {
		return mdk.StepResult{Err: fmt.Errorf("code required")}
	}
	t, err := m.GetTaxonomy(sCtx.Ctx, code)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"taxonomy": t}}
}

func (m *Module) GetTermBySlugStep(sCtx mdk.StepContext) mdk.StepResult {
	slug, _ := sCtx.Input["slug"].(string)
	if slug == "" {
		return mdk.StepResult{Err: fmt.Errorf("slug required")}
	}
	t, err := m.GetTerm(sCtx.Ctx, slug)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"term": t}}
}

func (m *Module) CreateTaxonomyStep(sCtx mdk.StepContext) mdk.StepResult {
	ba, err := json.Marshal(sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	var in CreateTaxonomyInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return mdk.StepResult{Err: err}
	}

	tax, err := m.CreateTaxonomy(sCtx.Ctx, in)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"taxonomy": tax}}
}

func (m *Module) CreateTermStep(sCtx mdk.StepContext) mdk.StepResult {
	ba, err := json.Marshal(sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	var in CreateTaxonomyTermInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return mdk.StepResult{Err: err}
	}

	term, err := m.CreateTaxonomyTerm(sCtx.Ctx, in)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"term": term}}
}

func (m *Module) LinkStep(sCtx mdk.StepContext) mdk.StepResult {
	ba, err := json.Marshal(sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	var in LinkResourceInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return mdk.StepResult{Err: err}
	}

	ok, err := m.LinkResource(sCtx.Ctx, in)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	return mdk.StepResult{Output: map[string]any{"linked": ok}}
}

// Shutdown performs cleanups when shutting down hyperrr.
func (m *Module) Shutdown(ctx context.Context) error {
	return nil
}

// Models returns instances of the taxonomy database schemas to migrate.
func (m *Module) Models() []any {
	return []any{&Taxonomy{}, &TaxonomyTerm{}, &TaxonomyRelation{}}
}

// Routes returns HTTP endpoints (none for this module).
func (m *Module) Routes() []mdk.Route {
	return nil
}

// Repo returns the taxonomy repository.
func (m *Module) Repo() *Repository {
	return m.repo
}

// ListResources implements MCP list resources.
func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	taxonomies, err := m.repo.ListTaxonomies(ctx)
	if err != nil {
		return nil, err
	}
	var res []mdk.MCPResource
	for _, t := range taxonomies {
		res = append(res, mdk.MCPResource{
			URI:         "taxonomy://" + t.Code,
			Name:        "Taxonomy: " + t.Name,
			Description: "Hierarchy and terms for taxonomy " + t.Code,
			MimeType:    "application/json",
		})
	}
	return res, nil
}

// ReadResource implements MCP read resource.
func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	var code string
	if n, err := fmt.Sscanf(uri, "taxonomy://%s", &code); err != nil || n != 1 {
		return "", fmt.Errorf("invalid taxonomy URI")
	}
	t, err := m.repo.GetTaxonomyByCode(ctx, code)
	if err != nil {
		return "", err
	}
	dataBytes, err := json.Marshal(t)
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
