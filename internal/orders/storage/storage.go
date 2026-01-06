// internal/orders/storage/repository.go
package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/orders/model"
	"github.com/google/uuid"
)

type Repository interface {
	// Методы для gophermart
	SaveOrder(ctx context.Context, order *model.Order) error
	OrderExists(ctx context.Context, orderNumber string) (bool, error)
	GetOrderByNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.Order, error)

	// Методы для accrual
	GetOrderForRegistration(ctx context.Context, orderNumber string) (*model.Order, error)
	UpdateOrderToRegistered(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error

	// Методы для воркера
	SaveOrderItems(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) error
	GetOrdersByStatus(ctx context.Context, status string, limit int) ([]model.Order, error)
	GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error)
	UpdateOrderWithAccrual(ctx context.Context, orderID uuid.UUID, status string, accrual float64) error
	UpdateOrderUser(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) error
}
