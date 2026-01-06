package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/rewards/model"
)

type Repository interface {
	Create(ctx context.Context, reward *model.Reward) error

	FindByMatch(ctx context.Context, match string) (*model.Reward, error)

	FindAll(ctx context.Context) ([]model.Reward, error)
}
