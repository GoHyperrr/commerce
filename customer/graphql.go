package customer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Queries exposes resolver query maps.
func (m *Module) Queries() map[string]any {
	return map[string]any{
		"getCustomer":          m.GetCustomer,
		"listCustomers":        m.ListCustomers,
		"getCustomerAddresses": m.GetCustomerAddresses,
	}
}

// Mutations exposes resolver mutation maps.
func (m *Module) Mutations() map[string]any {
	return map[string]any{
		"createCustomer":        m.CreateCustomer,
		"updateCustomer":        m.UpdateCustomer,
		"deleteCustomer":        m.DeleteCustomer,
		"addCustomerAddress":    m.AddCustomerAddress,
		"updateCustomerAddress": m.UpdateCustomerAddress,
		"deleteCustomerAddress": m.DeleteCustomerAddress,
	}
}

// FieldResolvers exposes custom field-level resolvers.
func (m *Module) FieldResolvers() map[string]any {
	return nil
}

// GetCustomer retrieves an individual customer by ID.
func (m *Module) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	return m.repo.GetByID(ctx, id)
}

// ListCustomers retrieves a list of all customer profiles.
func (m *Module) ListCustomers(ctx context.Context) ([]*Customer, error) {
	return m.repo.List(ctx)
}

// GetCustomerAddresses lists all addresses belonging to a customer.
func (m *Module) GetCustomerAddresses(ctx context.Context, customerID string) ([]*Address, error) {
	return m.repo.GetAddressesByCustomerID(ctx, customerID)
}

// CreateCustomer creates a new customer profile.
func (m *Module) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*Customer, error) {
	cID := "cust_" + uuid.New().String()
	
	var isGuest bool
	if input.IsGuest != nil {
		isGuest = *input.IsGuest
	}

	var uID string
	if input.UserID != nil {
		uID = *input.UserID
	}

	var phone string
	if input.Phone != nil {
		phone = *input.Phone
	}

	c := &Customer{
		ID:       cID,
		UserID:   uID,
		IsGuest:  isGuest,
		Name:     input.Name,
		Email:    input.Email,
		Phone:    phone,
		Metadata: input.Metadata,
	}

	err := m.repo.Save(ctx, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateCustomer updates customer profile details and dynamically updates default addresses.
func (m *Module) UpdateCustomer(ctx context.Context, id string, input UpdateCustomerInput) (*Customer, error) {
	c, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		c.Name = *input.Name
	}
	if input.Email != nil {
		c.Email = *input.Email
	}
	if input.Phone != nil {
		c.Phone = *input.Phone
	}
	if input.DefaultShippingAddressID != nil {
		c.DefaultShippingAddressID = input.DefaultShippingAddressID
	}
	if input.DefaultBillingAddressID != nil {
		c.DefaultBillingAddressID = input.DefaultBillingAddressID
	}
	if input.Metadata != nil {
		c.Metadata = input.Metadata
	}

	err = m.repo.Save(ctx, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// DeleteCustomer deletes a customer record.
func (m *Module) DeleteCustomer(ctx context.Context, id string) (bool, error) {
	err := m.repo.Delete(ctx, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

// AddCustomerAddress adds a shipping/billing address to a customer profile.
func (m *Module) AddCustomerAddress(ctx context.Context, customerID string, input CreateAddressInput) (*Address, error) {
	_, err := m.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found")
	}

	var recName string
	if input.ReceiverName != nil {
		recName = *input.ReceiverName
	}
	var phone string
	if input.Phone != nil {
		phone = *input.Phone
	}
	var line2 string
	if input.Line2 != nil {
		line2 = *input.Line2
	}

	addr := &Address{
		ID:           "addr_" + uuid.New().String(),
		CustomerID:   customerID,
		ReceiverName: recName,
		Phone:        phone,
		Line1:        input.Line1,
		Line2:        line2,
		City:         input.City,
		State:        input.State,
		Zip:          input.Zip,
		Country:      input.Country,
	}

	err = m.repo.SaveAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// UpdateCustomerAddress updates address record values.
func (m *Module) UpdateCustomerAddress(ctx context.Context, addressID string, input UpdateAddressInput) (*Address, error) {
	addr, err := m.repo.GetAddressByID(ctx, addressID)
	if err != nil {
		return nil, err
	}

	if input.ReceiverName != nil {
		addr.ReceiverName = *input.ReceiverName
	}
	if input.Phone != nil {
		addr.Phone = *input.Phone
	}
	if input.Line1 != nil {
		addr.Line1 = *input.Line1
	}
	if input.Line2 != nil {
		addr.Line2 = *input.Line2
	}
	if input.City != nil {
		addr.City = *input.City
	}
	if input.State != nil {
		addr.State = *input.State
	}
	if input.Zip != nil {
		addr.Zip = *input.Zip
	}
	if input.Country != nil {
		addr.Country = *input.Country
	}

	err = m.repo.SaveAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// DeleteCustomerAddress deletes an address record.
func (m *Module) DeleteCustomerAddress(ctx context.Context, addressID string) (bool, error) {
	err := m.repo.DeleteAddress(ctx, addressID)
	if err != nil {
		return false, err
	}
	return true, nil
}
