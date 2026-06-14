package payments

import (
	"time"

	"gorm.io/gorm"
)

type TransactionStatus string

const (
	StatusPending   TransactionStatus = "PENDING"
	StatusSucceeded TransactionStatus = "SUCCEEDED"
	StatusFailed    TransactionStatus = "FAILED"
	StatusRefunded  TransactionStatus = "REFUNDED"
)

// PaymentTransaction tracks payment gateway state changes linked to orders.
type PaymentTransaction struct {
	ID                    string            `gorm:"primaryKey" json:"id"`
	OrderID               string            `gorm:"index" json:"order_id"`
	Provider              string            `json:"provider"` // "stripe", "razorpay", "mock"
	ProviderTransactionID string            `gorm:"index" json:"provider_transaction_id"`
	Amount                float64           `json:"amount"`
	Currency              string            `json:"currency"`
	Status                TransactionStatus `gorm:"not null" json:"status"`
	ClientSecret          string            `json:"client_secret,omitempty"` // Stripe client_secret or Razorpay order_id
	Metadata              string            `json:"metadata,omitempty"`      // JSON metadata string
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	DeletedAt             gorm.DeletedAt    `gorm:"index" json:"-"`
}
