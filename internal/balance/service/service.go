// internal/balance/service/service.go
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/as-tanais/hofermart/internal/balance/model"
	"github.com/as-tanais/hofermart/internal/balance/storage"
	"github.com/as-tanais/hofermart/internal/orders"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrInvalidOrderNumber = errors.New("invalid order number")
	ErrOrderAlreadyExists = errors.New("order already exists")
)

type BalanceService struct {
	repo storage.Repository
	log  *zap.Logger
}

func NewBalanceService(repo storage.Repository, log *zap.Logger) *BalanceService {
	return &BalanceService{
		repo: repo,
		log:  log,
	}
}

// GetBalance возвращает баланс пользователя
func (s *BalanceService) GetBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	balance, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get user balance",
			zap.Stringer("userID", userID),
			zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved user balance",
		zap.Stringer("userID", userID),
		zap.Float64("current", balance.Current),
		zap.Float64("withdrawn", balance.Withdrawn))

	return balance, nil
}

// Withdraw списывает средства с баланса
func (s *BalanceService) Withdraw(ctx context.Context, userID uuid.UUID, order string, sum float64) error {
	// 1. Валидация номера заказа (алгоритм Луна)
	if !orders.IsValidLuhn(order) {
		s.log.Warn("Invalid order number (Luhn check failed)",
			zap.String("order", order),
			zap.Stringer("userID", userID))
		return ErrInvalidOrderNumber
	}

	// 2. Проверяем положительную сумму
	if sum <= 0 {
		s.log.Warn("Invalid withdrawal amount",
			zap.Float64("sum", sum),
			zap.Stringer("userID", userID))
		return errors.New("amount must be positive")
	}

	// 3. Создаем списание
	err := s.repo.CreateWithdrawal(ctx, userID, order, sum)
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "insufficient funds") {
			s.log.Warn("Insufficient funds for withdrawal",
				zap.Stringer("userID", userID),
				zap.Float64("sum", sum))
			return ErrInsufficientFunds
		}

		if strings.Contains(errMsg, "order number already exists") {
			s.log.Warn("Order number already exists",
				zap.String("order", order),
				zap.Stringer("userID", userID))
			return ErrOrderAlreadyExists
		}

		s.log.Error("Failed to create withdrawal",
			zap.Stringer("userID", userID),
			zap.String("order", order),
			zap.Float64("sum", sum),
			zap.Error(err))
		return err
	}

	s.log.Info("Successful withdrawal",
		zap.Stringer("userID", userID),
		zap.String("order", order),
		zap.Float64("sum", sum))

	return nil
}

// GetUserWithdrawals возвращает историю списаний
func (s *BalanceService) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.Withdrawal, error) {
	withdrawals, err := s.repo.GetUserWithdrawals(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get user withdrawals",
			zap.Stringer("userID", userID),
			zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved user withdrawals",
		zap.Stringer("userID", userID),
		zap.Int("count", len(withdrawals)))

	return withdrawals, nil
}
