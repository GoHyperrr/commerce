package customer

import (
	"time"

	"github.com/GoHyperrr/mdk"
	"gorm.io/gorm"
)

// Customer represents a customer profile.
type Customer struct {
	ID                       string         `gorm:"primaryKey" json:"id"`
	UserID                   string         `gorm:"index" json:"userId"` // Links to user account in 'auth' module (empty for guests)
	IsGuest                  bool           `gorm:"default:false" json:"isGuest"`
	Name                     string         `json:"name"`
	Email                    string         `json:"email"`
	Phone                    string         `json:"phone"`
	DefaultShippingAddressID *string        `json:"defaultShippingAddressId"`
	DefaultBillingAddressID  *string        `json:"defaultBillingAddressId"`
	Addresses                []Address      `gorm:"foreignKey:CustomerID;constraint:OnDelete:CASCADE" json:"addresses"`
	Metadata                 mdk.Metadata   `gorm:"type:text" json:"metadata"`
	CreatedAt                time.Time      `json:"createdAt"`
	UpdatedAt                time.Time      `json:"updatedAt"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`
}

// Address represents a customer's physical shipping or billing location.
type Address struct {
	ID           string         `gorm:"primaryKey" json:"id"`
	CustomerID   string         `gorm:"index;not null" json:"customerId"`
	ReceiverName string         `json:"receiverName"` // Recipient name
	Phone        string         `json:"phone"`        // Contact number for delivery
	Line1        string         `json:"line1"`
	Line2        string         `json:"line2"`
	City         string         `json:"city"`
	State        string         `json:"state"`
	Zip          string         `json:"zip"`
	Country      string         `json:"country"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Input mapping structures for GraphQL endpoints.
type CreateCustomerInput struct {
	UserID   *string      `json:"userId"`
	IsGuest  *bool        `json:"isGuest"`
	Name     string       `json:"name"`
	Email    string       `json:"email"`
	Phone    *string      `json:"phone"`
	Metadata mdk.Metadata `json:"metadata"`
}

type UpdateCustomerInput struct {
	Name                     *string      `json:"name"`
	Email                    *string      `json:"email"`
	Phone                    *string      `json:"phone"`
	DefaultShippingAddressID *string      `json:"defaultShippingAddressId"`
	DefaultBillingAddressID  *string      `json:"defaultBillingAddressId"`
	Metadata                 mdk.Metadata `json:"metadata"`
}

type CreateAddressInput struct {
	ReceiverName *string `json:"receiverName"`
	Phone        *string `json:"phone"`
	Line1        string  `json:"line1"`
	Line2        *string `json:"line2"`
	City         string  `json:"city"`
	State        string  `json:"state"`
	Zip          string  `json:"zip"`
	Country      string  `json:"country"`
}

type UpdateAddressInput struct {
	ReceiverName *string `json:"receiverName"`
	Phone        *string `json:"phone"`
	Line1        *string `json:"line1"`
	Line2        *string `json:"line2"`
	City         *string `json:"city"`
	State        *string `json:"state"`
	Zip          *string `json:"zip"`
	Country      *string `json:"country"`
}
