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

// RegisterOrder - регистрация заказа пользователем в Gophermart
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

	// 3. Создаем заказ со статусом REGISTERED
	order := &model.Order{
		ID:          uuid.New(),
		UserID:      userID,
		OrderNumber: req.OrderNumber,
		Status:      "REGISTERED", // ← Для Gophermart тоже REGISTERED
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

// RegisterOrderWithGoods - регистрирует заказ с товарами от системы accrual
func (s *Service) RegisterOrderWithGoods(ctx context.Context, req *dto.AccrualOrderRequest) error {
	// 1. Проверяем валидность номера заказа
	if !orders.IsValidLuhn(req.Order) {
		return fmt.Errorf("invalid order number")
	}

	// 2. Проверяем существование заказа
	order, err := s.repo.GetOrderByNumber(ctx, req.Order)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		// Создаем новый заказ без пользователя
		order = &model.Order{
			ID:          uuid.New(),
			OrderNumber: req.Order,
			Status:      "REGISTERED", // ← Accrual статус!
		}

		if err := s.repo.SaveOrder(ctx, order); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}
	} else {
		// Если заказ уже существует, проверяем статус
		// Если уже обрабатывается или обработан - ошибка
		if order.Status == "PROCESSING" || order.Status == "PROCESSED" || order.Status == "INVALID" {
			return fmt.Errorf("order already processed")
		}
		// Если статус REGISTERED - можно обновить товары
	}

	// 3. Сохраняем/обновляем товары
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

	s.logger.Info("Order registered with goods from accrual",
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
