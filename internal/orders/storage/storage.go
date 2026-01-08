package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/orders/model"
	"github.com/google/uuid"
)

type Repository interface {
	// Базовые операции
	SaveOrder(ctx context.Context, order *model.Order) error
	OrderExists(ctx context.Context, orderNumber string) (bool, error)
	FindByNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.Order, error)
	UpdateOrderUser(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) error
	UpdateOrderUserAndStatus(ctx context.Context, orderID, userID uuid.UUID, status string) error

	// Методы для accrual
	UpdateOrderToRegistered(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error

	// воркер
	SaveOrderItems(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error
	GetOrdersByStatus(ctx context.Context, status string, limit int) ([]model.Order, error)
	GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error)

	Update(ctx context.Context, order *model.Order) error
}
