package model

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	OrderNumber string
	Status      string
	Items       []OrderItem
	Accrual     *float64
	CreatedAt   time.Time
}

type OrderItem struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	Description string
	Price       float64
}
