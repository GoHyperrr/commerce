package cart

import (
	"time"

	"github.com/GoHyperrr/commerce/customer"
	"gorm.io/gorm"
)

type CheckoutInput struct {
	ShippingAddressID   *string                      `json:"shippingAddressId"`
	BillingAddressID    *string                      `json:"billingAddressId"`
	ShippingAddress     *customer.CreateAddressInput `json:"shippingAddress"`
	BillingAddress      *customer.CreateAddressInput `json:"billingAddress"`
	PaymentMethod       *string                      `json:"paymentMethod"`
	PaymentDetails      map[string]any               `json:"paymentDetails"`
	ShippingCarrier     *string                      `json:"shippingCarrier"`
	ShippingMethod      *string                      `json:"shippingMethod"`
}

type CartStatus string

const (
	CartActive    CartStatus = "ACTIVE"
	CartCompleted CartStatus = "COMPLETED"
	CartAbandoned CartStatus = "ABANDONED"
)

// Cart represents a temporary shopping session.
type Cart struct {
	ID         string         `gorm:"primaryKey" json:"id"`
	CustomerID          string         `gorm:"index" json:"customer_id"`
	Status              CartStatus     `gorm:"not null" json:"status"`
	ShippingAddressID   *string        `json:"shipping_address_id"`
	BillingAddressID    *string        `json:"billing_address_id"`
	ShippingAddressJSON *string        `json:"shipping_address_json"`
	BillingAddressJSON  *string        `json:"billing_address_json"`
	PaymentMethod       string         `json:"payment_method"`
	ShippingCarrier     string         `json:"shipping_carrier"`
	ShippingMethod      string         `json:"shipping_method"`
	Items               []CartItem     `gorm:"foreignKey:CartID" json:"items"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// CartItem represents an item within a cart.
type CartItem struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	CartID    string         `gorm:"index" json:"cart_id"`
	ProductID string         `json:"product_id"`
	Quantity  int            `json:"quantity"`
	Price     float64        `json:"price"` // Price at the time of adding to cart
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type AddItemInput struct {
	ProductID string  `json:"productId"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

