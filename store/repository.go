package store

import (
	"context"

	"gorm.io/gorm"
)

// Repository handles database interactions for the singleton StoreSettings.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new database repository instance.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Get retrieves the singleton StoreSettings (ID = 1).
func (r *Repository) Get(ctx context.Context) (*StoreSettings, error) {
	var settings StoreSettings
	err := r.db.WithContext(ctx).First(&settings, 1).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Save saves or updates the StoreSettings entity.
func (r *Repository) Save(ctx context.Context, settings *StoreSettings) error {
	settings.ID = 1 // Force ID to 1 to ensure singleton pattern
	return r.db.WithContext(ctx).Save(settings).Error
}
