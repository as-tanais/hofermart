package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/balance/model"
	"github.com/google/uuid"
)

type Repository interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error)
	CreateWithdrawal(ctx context.Context, userID uuid.UUID, orderNumber string, sum float64) error
	GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.Withdrawal, error)
	CheckOrderExists(ctx context.Context, orderNumber string) (bool, error)
}
