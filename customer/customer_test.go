package customer

import (
	"context"
	"testing"

	"github.com/GoHyperrr/mdk"
	"github.com/GoHyperrr/mdk/mdktest"
)

func TestCustomerModule(t *testing.T) {
	rt, _ := mdktest.NewInMemoryTestRuntime()

	mod := NewModule()
	_ = rt.DB().AutoMigrate(mod.Models()...)

	if err := mod.Init(context.Background(), rt); err != nil {
		t.Fatalf("failed to initialize customer module: %v", err)
	}

	var customerID string
	var guestID string
	var addrID string

	t.Run("Create Registered Customer", func(t *testing.T) {
		uID := "auth_user_123"
		phoneVal := "1234567890"
		input := CreateCustomerInput{
			UserID: &uID,
			Name:   "Alice Smith",
			Email:  "alice@example.com",
			Phone:  &phoneVal,
		}

		c, err := mod.CreateCustomer(context.Background(), input)
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}
		if c.UserID != "auth_user_123" || c.IsGuest {
			t.Errorf("unexpected customer profile: %+v", c)
		}
		customerID = c.ID
	})

	t.Run("Create Guest Customer", func(t *testing.T) {
		isGuest := true
		input := CreateCustomerInput{
			IsGuest: &isGuest,
			Name:    "Guest User",
			Email:   "guest@example.com",
		}

		c, err := mod.CreateCustomer(context.Background(), input)
		if err != nil {
			t.Fatalf("failed to create guest customer: %v", err)
		}
		if !c.IsGuest || c.UserID != "" {
			t.Errorf("expected guest profile, got: %+v", c)
		}
		guestID = c.ID
	})

	t.Run("Add Customer Address", func(t *testing.T) {
		recName := "Alice Smith"
		phone := "1234567890"
		line2 := "Apt 4B"

		input := CreateAddressInput{
			ReceiverName: &recName,
			Phone:        &phone,
			Line1:        "123 Main St",
			Line2:        &line2,
			City:         "Metropolis",
			State:        "NY",
			Zip:          "10001",
			Country:      "US",
		}

		addr, err := mod.AddCustomerAddress(context.Background(), customerID, input)
		if err != nil {
			t.Fatalf("failed to add address: %v", err)
		}
		if addr.CustomerID != customerID || addr.ReceiverName != "Alice Smith" || addr.Line2 != "Apt 4B" {
			t.Errorf("unexpected address: %+v", addr)
		}
		addrID = addr.ID
	})

	t.Run("Update Customer and Set Default Address", func(t *testing.T) {
		updatedName := "Alice J. Smith"
		input := UpdateCustomerInput{
			Name:                     &updatedName,
			DefaultShippingAddressID: &addrID,
			DefaultBillingAddressID:  &addrID,
		}

		c, err := mod.UpdateCustomer(context.Background(), customerID, input)
		if err != nil {
			t.Fatalf("failed to update customer: %v", err)
		}
		if c.Name != "Alice J. Smith" || c.DefaultShippingAddressID == nil || *c.DefaultShippingAddressID != addrID {
			t.Errorf("unexpected updated customer profile: %+v", c)
		}
	})

	t.Run("Query Customer Details & Addresses", func(t *testing.T) {
		c, err := mod.GetCustomer(context.Background(), customerID)
		if err != nil {
			t.Fatalf("failed to query customer: %v", err)
		}
		if len(c.Addresses) != 1 || c.Addresses[0].ID != addrID {
			t.Errorf("expected 1 address, got %+v", c.Addresses)
		}

		addrs, err := mod.GetCustomerAddresses(context.Background(), customerID)
		if err != nil {
			t.Fatalf("failed to query addresses: %v", err)
		}
		if len(addrs) != 1 || addrs[0].ID != addrID {
			t.Errorf("expected address list matching %s, got %+v", addrID, addrs)
		}
	})

	t.Run("Update Customer Address", func(t *testing.T) {
		newLine1 := "456 Oak Ave"
		input := UpdateAddressInput{
			Line1: &newLine1,
		}

		addr, err := mod.UpdateCustomerAddress(context.Background(), addrID, input)
		if err != nil {
			t.Fatalf("failed to update address: %v", err)
		}
		if addr.Line1 != "456 Oak Ave" {
			t.Errorf("expected updated address Line1 to be '456 Oak Ave', got %s", addr.Line1)
		}
	})

	t.Run("Delete Address", func(t *testing.T) {
		deleted, err := mod.DeleteCustomerAddress(context.Background(), addrID)
		if err != nil || !deleted {
			t.Fatalf("failed to delete address: %v", err)
		}

		addrs, err := mod.GetCustomerAddresses(context.Background(), customerID)
		if err != nil {
			t.Fatalf("failed to query address list: %v", err)
		}
		if len(addrs) != 0 {
			t.Errorf("expected 0 addresses, got %d", len(addrs))
		}
	})

	t.Run("Delete Customer", func(t *testing.T) {
		deleted, err := mod.DeleteCustomer(context.Background(), guestID)
		if err != nil || !deleted {
			t.Fatalf("failed to delete customer: %v", err)
		}

		c, err := mod.GetCustomer(context.Background(), guestID)
		if err == nil {
			t.Errorf("expected customer to be deleted, found record: %+v", c)
		}
	})

	t.Run("Identity Event Seeding Subscription", func(t *testing.T) {
		evt := mdk.Event{
			Namespace: "identity",
			Type:      "user_created",
			Payload: map[string]any{
				"actor_id": "actor_test_1",
				"user_id":  "user_test_1",
				"name":     "Test Seeding",
				"email":    "seeding@example.com",
			},
		}

		err := rt.Bus().Publish(context.Background(), evt)
		if err != nil {
			t.Fatalf("failed to publish event: %v", err)
		}

		// Verify that a customer profile was seeded
		c, err := mod.Repo().GetByID(context.Background(), "cust_user_test_1")
		if err != nil {
			t.Fatalf("failed to fetch seeded customer: %v", err)
		}
		if c.Name != "Test Seeding" || c.UserID != "actor_test_1" {
			t.Errorf("unexpected seeded customer profile: %+v", c)
		}
	})
}
