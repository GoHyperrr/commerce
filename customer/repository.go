package customer

import (
	"context"

	"gorm.io/gorm"
)

// Repository handles database interactions for Customers and Addresses.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new customer Repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// Save persists a customer record.
func (r *Repository) Save(ctx context.Context, c *Customer) error {
	return r.db.WithContext(ctx).Save(c).Error
}

// GetByID retrieves a customer profile by its unique ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Customer, error) {
	var c Customer
	err := r.db.WithContext(ctx).Preload("Addresses").First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByUserID retrieves a customer profile by its user identifier.
func (r *Repository) GetByUserID(ctx context.Context, userID string) (*Customer, error) {
	var c Customer
	err := r.db.WithContext(ctx).Preload("Addresses").First(&c, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List lists all customer profiles in the database.
func (r *Repository) List(ctx context.Context) ([]*Customer, error) {
	var list []*Customer
	err := r.db.WithContext(ctx).Preload("Addresses").Find(&list).Error
	return list, err
}

// Delete removes a customer profile from the database.
func (r *Repository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&Customer{}, "id = ?", id).Error
}

// SaveAddress saves or updates an address record.
func (r *Repository) SaveAddress(ctx context.Context, addr *Address) error {
	return r.db.WithContext(ctx).Save(addr).Error
}

// GetAddressByID retrieves a specific address by its ID.
func (r *Repository) GetAddressByID(ctx context.Context, id string) (*Address, error) {
	var addr Address
	err := r.db.WithContext(ctx).First(&addr, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &addr, nil
}

// DeleteAddress deletes an address record.
func (r *Repository) DeleteAddress(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&Address{}, "id = ?", id).Error
}

// GetAddressesByCustomerID lists all addresses registered under a customer.
func (r *Repository) GetAddressesByCustomerID(ctx context.Context, customerID string) ([]*Address, error) {
	var list []*Address
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).Find(&list).Error
	return list, err
}
