package product

import (
	"context"

	"gorm.io/gorm"
)

// Repository handles database operations for products, variants, and catalog entities.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new product repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// Save persists a product and all of its nested relationships (options, variants, images).
func (r *Repository) Save(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(p).Error
}

// GetByID retrieves a product by its ID, preloading all options, variants, options per variant, and images.
func (r *Repository) GetByID(ctx context.Context, id string) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).
		Preload("Options").
		Preload("Variants").
		Preload("Variants.Options").
		Preload("Images").
		First(&p, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByHandle retrieves a product by its unique URL slug/handle, preloading all relationships.
func (r *Repository) GetByHandle(ctx context.Context, handle string) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).
		Preload("Options").
		Preload("Variants").
		Preload("Variants.Options").
		Preload("Images").
		First(&p, "handle = ?", handle).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List retrieves all products with all of their relationships preloaded.
func (r *Repository) List(ctx context.Context) ([]*Product, error) {
	var products []*Product
	err := r.db.WithContext(ctx).
		Preload("Options").
		Preload("Variants").
		Preload("Variants.Options").
		Preload("Images").
		Find(&products).Error
	return products, err
}

// Delete removes a product from the database (cascade deletes options, variants, images).
func (r *Repository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&Product{}, "id = ?", id).Error
}
