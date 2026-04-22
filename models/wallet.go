package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// UserWallet represents a user's digital wallet.
type UserWallet struct {
	ID        uint            `gorm:"primarykey" json:"id"`
	UserID    string          `gorm:"uniqueIndex;not null;size:64" json:"user_id"`
	Balance   decimal.Decimal `gorm:"type:numeric(20,2);default:0;not null" json:"balance"`
	Currency  string          `gorm:"size:3;default:'USD';not null" json:"currency"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
