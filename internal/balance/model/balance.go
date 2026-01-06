// internal/balance/model/balance.go
package model

import "github.com/google/uuid"

type Balance struct {
	UserID    uuid.UUID `json:"-"`
	Current   float64   `json:"current"`
	Withdrawn float64   `json:"withdrawn"`
}

type Withdrawal struct {
	ID          uuid.UUID `json:"-" db:"id"`
	UserID      uuid.UUID `json:"-" db:"user_id"`
	Order       string    `json:"order" db:"order_number"`
	Sum         float64   `json:"sum" db:"sum"`
	ProcessedAt string    `json:"processed_at" db:"processed_at"`
}
