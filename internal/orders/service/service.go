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
	if err := s.validateOrderNumber(req.OrderNumber); err != nil {
		return err
	}

	s.logger.Info("RegisterOrder: processing",
		zap.String("userID", userID.String()),
		zap.String("order", req.OrderNumber))

	existingOrder, err := s.repo.FindByNumber(ctx, req.OrderNumber)
	if err != nil {
		s.logger.Error("Failed to check order", zap.Error(err))
		return fmt.Errorf("failed to check order: %w", err)
	}

	if existingOrder != nil {
		s.logger.Info("Order exists",
			zap.String("order", req.OrderNumber),
			zap.String("status", existingOrder.Status),
			zap.String("existingUserID", existingOrder.UserID.String()),
			zap.String("currentUserID", userID.String()))

		if existingOrder.Status == "REGISTERED" && existingOrder.UserID == uuid.Nil {
			s.logger.Info("Attaching REGISTERED order to user, changing to NEW")

			if err := s.repo.UpdateOrderUserAndStatus(ctx, existingOrder.ID, userID, "NEW"); err != nil {
				s.logger.Error("Failed to update order", zap.Error(err))
				return err
			}
			return nil
		}

		if existingOrder.Status == "NEW" {
			if existingOrder.UserID == userID {
				s.logger.Info("Order already in NEW status for same user")
				return orders.ErrOrderExistsSameUser
			}
			s.logger.Warn("Order in NEW status but belongs to other user")
			return orders.ErrOrderExistsOtherUser
		}

		s.logger.Warn("Order exists in non-NEW status",
			zap.String("status", existingOrder.Status))

		if existingOrder.UserID == userID {
			return orders.ErrOrderExistsSameUser
		}
		return orders.ErrOrderExistsOtherUser
	}

	s.logger.Info("Creating new order with status NEW")

	order := &model.Order{
		ID:          uuid.New(),
		UserID:      userID,
		OrderNumber: req.OrderNumber,
		Status:      "NEW",
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		s.logger.Error("Failed to save order", zap.Error(err))
		return err
	}

	s.logger.Info("New order created successfully")
	return nil
}

func (s *Service) GetOrder(ctx context.Context, orderNumber string) (*model.Order, error) {
	if !orders.IsValidLuhn(orderNumber) {
		return nil, orders.ErrInvalidData
	}

	order, err := s.repo.FindByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	return order, nil
}

func (s *Service) RegisterOrderWithGoods(ctx context.Context, req *dto.AccrualOrderRequest) error {
	if !orders.IsValidLuhn(req.Order) {
		return fmt.Errorf("invalid order number")
	}

	order, err := s.repo.FindByNumber(ctx, req.Order)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		order = &model.Order{
			ID:          uuid.New(),
			OrderNumber: req.Order,
			Status:      "REGISTERED",
		}

		if err := s.repo.SaveOrder(ctx, order); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}
	} else {
		if order.Status == "PROCESSING" || order.Status == "PROCESSED" || order.Status == "INVALID" {
			return fmt.Errorf("order already processed")
		}
	}

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
	return s.repo.FindByNumber(ctx, orderNumber)
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
