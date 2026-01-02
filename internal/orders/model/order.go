package model

import "github.com/google/uuid"

type Order struct {
	ID          uuid.UUID
	OrderNumber string
	Status      string
	Items       []OrderItem
	Accrual     *float64
}

type OrderItem struct {
	ID          uuid.UUID
	Description string
	Price       float64
}
