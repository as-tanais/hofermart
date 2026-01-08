package service

import (
	"context"
	"errors"
	"fmt"

	balerr "github.com/as-tanais/hofermart/internal/balance"
	"github.com/as-tanais/hofermart/internal/balance/model"
	"github.com/as-tanais/hofermart/internal/balance/storage"
	"github.com/as-tanais/hofermart/internal/orders"
	"github.com/google/uuid"
	"go.uber.org/zap"
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

func (s *BalanceService) GetBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	balance, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get user balance",
			zap.Stringer("userID", userID),
			zap.Error(err))

		if errors.Is(err, balerr.ErrDatabaseError) {
			return nil, fmt.Errorf("database error while fetching balance")
		}
		return nil, err
	}

	s.log.Debug("Retrieved user balance",
		zap.Stringer("userID", userID),
		zap.Float64("current", balance.Current),
		zap.Float64("withdrawn", balance.Withdrawn))

	return balance, nil
}

func (s *BalanceService) Withdraw(ctx context.Context, userID uuid.UUID, order string, sum float64) error {

	if !orders.IsValidLuhn(order) {
		s.log.Warn("Invalid order number (Luhn check failed)",
			zap.String("order", order),
			zap.Stringer("userID", userID))
		return balerr.ErrInvalidOrderNumber
	}

	if sum <= 0 {
		s.log.Warn("Invalid withdrawal amount",
			zap.Float64("sum", sum),
			zap.Stringer("userID", userID))
		return balerr.ErrNegativeAmount
	}

	err := s.repo.CreateWithdrawal(ctx, userID, order, sum)
	if err != nil {

		if errors.Is(err, balerr.ErrInsufficientFunds) {
			s.log.Warn("Insufficient funds for withdrawal",
				zap.Stringer("userID", userID),
				zap.Float64("sum", sum),
				zap.Error(err))
			return balerr.ErrInsufficientFunds
		}

		if errors.Is(err, balerr.ErrOrderAlreadyExists) {
			s.log.Warn("Order number already exists",
				zap.String("order", order),
				zap.Stringer("userID", userID),
				zap.Error(err))
			return balerr.ErrOrderAlreadyExists
		}

		if errors.Is(err, balerr.ErrDatabaseError) {
			s.log.Error("Database error during withdrawal",
				zap.Stringer("userID", userID),
				zap.String("order", order),
				zap.Float64("sum", sum),
				zap.Error(err))
			return fmt.Errorf("database error while processing withdrawal")
		}

		s.log.Error("Failed to create withdrawal",
			zap.Stringer("userID", userID),
			zap.String("order", order),
			zap.Float64("sum", sum),
			zap.Error(err))
		return fmt.Errorf("withdrawal failed: %w", err)
	}

	s.log.Info("Successful withdrawal",
		zap.Stringer("userID", userID),
		zap.String("order", order),
		zap.Float64("sum", sum))

	return nil
}

func (s *BalanceService) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.Withdrawal, error) {
	withdrawals, err := s.repo.GetUserWithdrawals(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get user withdrawals",
			zap.Stringer("userID", userID),
			zap.Error(err))

		if errors.Is(err, balerr.ErrDatabaseError) {
			return nil, fmt.Errorf("database error while fetching withdrawals")
		}
		return nil, err
	}

	s.log.Debug("Retrieved user withdrawals",
		zap.Stringer("userID", userID),
		zap.Int("count", len(withdrawals)))

	return withdrawals, nil
}
