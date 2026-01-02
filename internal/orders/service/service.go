package service

import (
	"context"

	"github.com/as-tanais/hofermart/internal/orders"
	"github.com/as-tanais/hofermart/internal/orders/dto"
	"github.com/as-tanais/hofermart/internal/orders/model"
	"github.com/as-tanais/hofermart/internal/orders/storage"
	"go.uber.org/zap"
)

type Service struct {
	repo   storage.Repository
	logger *zap.Logger
}

func NewService(repo storage.Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// internal/orders/service.go

func (s *Service) validateOrder(req *dto.CreateOrderReq) error {
	if req.OrderNumber == "" {
		return orders.ErrInvalidData
	}

	// 🔑 Проверка по алгоритму Луна
	if !orders.IsValidLuhn(req.OrderNumber) {
		return orders.ErrInvalidData
	}

	if len(req.Goods) == 0 {
		return orders.ErrInvalidData
	}
	for _, item := range req.Goods {
		if item.Description == "" || item.Price <= 0 {
			return orders.ErrInvalidData
		}
	}
	return nil
}

func (s *Service) RegisterOrder(ctx context.Context, req *dto.CreateOrderReq) error {
	if err := s.validateOrder(req); err != nil {
		return err
	}

	// Проверяем, не существует ли уже
	exists, err := s.repo.OrderExists(ctx, req.OrderNumber)
	if err != nil {
		s.logger.Error("Failed to check order existence", zap.Error(err))
		return err
	}
	if exists {
		return orders.ErrOrderExists
	}

	// Преобразуем в модель
	items := make([]model.OrderItem, len(req.Goods))
	for i, g := range req.Goods {
		items[i] = model.OrderItem{
			Description: g.Description,
			Price:       g.Price,
		}
	}

	order := &model.Order{
		OrderNumber: req.OrderNumber,
		Items:       items,
	}

	// Сохраняем
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		s.logger.Error("Failed to save order", zap.Error(err))
		return err
	}

	s.logger.Info("Order registered", zap.String("order_number", req.OrderNumber))
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
		return nil, nil // не найден
	}

	return order, nil
}
