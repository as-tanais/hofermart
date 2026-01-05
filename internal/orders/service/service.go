package service

import (
	"context"
	"fmt"

	"github.com/as-tanais/hofermart/internal/orders"
	"github.com/as-tanais/hofermart/internal/orders/dto"
	"github.com/as-tanais/hofermart/internal/orders/model"
	"github.com/as-tanais/hofermart/internal/orders/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repo   storage.Repository
	logger *zap.Logger
}

func NewService(repo storage.Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) validateOrderNumber(orderNumber string) error {
	if orderNumber == "" {
		return orders.ErrInvalidData
	}

	if !orders.IsValidLuhn(orderNumber) {
		return orders.ErrInvalidData
	}

	return nil
}

func (s *Service) RegisterOrder(ctx context.Context, userID uuid.UUID, req *dto.CreateOrderReq) error {
	// 1. Валидация номера заказа
	if err := s.validateOrderNumber(req.OrderNumber); err != nil {
		return err
	}

	// 2. Проверяем существование заказа
	existingOrder, err := s.repo.GetOrderByNumber(ctx, req.OrderNumber)
	if err != nil {
		s.logger.Error("Failed to check order existence", zap.Error(err))
		return fmt.Errorf("failed to check order: %w", err)
	}

	if existingOrder != nil {
		// Заказ уже существует
		if existingOrder.UserID == userID {
			// Этот же пользователь - возвращаем 200
			return orders.ErrOrderExistsSameUser
		} else {
			// Другой пользователь - конфликт 409
			return orders.ErrOrderExistsOtherUser
		}
	}

	// 3. Создаем заказ со статусом NEW (без товаров!)
	order := &model.Order{
		ID:          uuid.New(),
		UserID:      userID,
		OrderNumber: req.OrderNumber,
		Status:      "NEW", // Важно: заглавными буквами
		// Items НЕ заполняем - товары добавит accrual позже
	}

	// 4. Сохраняем заказ
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		s.logger.Error("Failed to save order", zap.Error(err))
		return err
	}

	s.logger.Info("Order registered by user",
		zap.String("userID", userID.String()),
		zap.String("order_number", req.OrderNumber))

	return nil
}

func (s *Service) GetOrder(ctx context.Context, orderNumber string) (*model.Order, error) {
	if !orders.IsValidLuhn(orderNumber) {
		return nil, orders.ErrInvalidData
	}

	order, err := s.repo.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	return order, nil
}

// RegisterOrderWithGoods - добавляет товары к заказу и меняет статус на REGISTERED
func (s *Service) RegisterOrderWithGoods(ctx context.Context, req *dto.AccrualOrderRequest) error {
	// 1. Находим заказ со статусом NEW
	order, err := s.repo.GetOrderByNumber(ctx, req.Order)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return fmt.Errorf("order not found: %s", req.Order)
	}

	if order.Status != "NEW" {
		return fmt.Errorf("order not found or not in NEW status")
	}

	// 2. Сохраняем товары
	items := make([]model.OrderItem, len(req.Goods))
	for i, good := range req.Goods {
		items[i] = model.OrderItem{
			ID:          uuid.New(),
			OrderID:     order.ID,
			Description: good.Description,
			Price:       good.Price,
		}
	}

	if err := s.repo.SaveOrderItems(ctx, order.ID, items); err != nil {
		return fmt.Errorf("failed to save order items: %w", err)
	}

	// 3. Меняем статус на REGISTERED
	if err := s.repo.UpdateOrderStatus(ctx, order.ID, "REGISTERED"); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	s.logger.Info("Order registered with goods",
		zap.String("order", req.Order),
		zap.Int("goods_count", len(items)))

	return nil
}

func (s *Service) GetOrderStatus(ctx context.Context, orderNumber string) (*model.Order, error) {
	return s.repo.GetOrderByNumber(ctx, orderNumber)
}

func (s *Service) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.Order, error) {
	orders, err := s.repo.GetUserOrders(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user orders",
			zap.String("userID", userID.String()),
			zap.Error(err))
		return nil, err
	}

	s.logger.Debug("Retrieved user orders",
		zap.String("userID", userID.String()),
		zap.Int("count", len(orders)))

	return orders, nil
}
