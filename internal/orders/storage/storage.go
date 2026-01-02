package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/orders/model"
)

type Repository interface {
	// SaveOrder сохраняет заказ и его товары.
	// Возвращает ошибку, если заказ с таким номером уже существует.
	SaveOrder(ctx context.Context, order *model.Order) error

	// OrderExists проверяет, существует ли заказ.
	OrderExists(ctx context.Context, orderNumber string) (bool, error)
	//Получение информации о расчёте начислений баллов лояльности за совершённый заказ.
	GetOrderByNumber(ctx context.Context, orderNumber string) (*model.Order, error)
}
