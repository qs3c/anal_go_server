package model

import (
	"time"
)

type Subscription struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	UserID        int64     `gorm:"not null;index" json:"user_id"`
	Plan          string    `gorm:"size:20;not null" json:"plan"` // basic, pro
	Amount        float64   `gorm:"type:decimal(10,2)" json:"amount,omitempty"`
	DailyQuota    int       `json:"daily_quota"`
	StartedAt     time.Time `gorm:"not null" json:"started_at"`
	ExpiresAt     time.Time `gorm:"not null;index" json:"expires_at"`
	Status        string    `gorm:"size:20;default:active;index" json:"status"` // active, expired, cancelled
	PaymentMethod string    `gorm:"size:20" json:"payment_method,omitempty"`    // wechat, alipay
	TransactionID string    `gorm:"size:100" json:"transaction_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}
