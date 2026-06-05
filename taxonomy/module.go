package taxonomy

import (
	"context"

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
	return nil
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

// ListResources implements MCP list resources (empty).
func (m *Module) ListResources(ctx context.Context) ([]mdk.MCPResource, error) {
	return nil, nil
}

// ReadResource implements MCP read resource (empty).
func (m *Module) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", nil
}

func init() {
	mdk.Register(func() mdk.Module {
		return NewModule()
	})
}
