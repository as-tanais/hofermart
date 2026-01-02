package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/rewards/model"
)

// Repository описывает операции с правилами вознаграждений.
type Repository interface {
	// Create сохраняет новое правило.
	// Возвращает ошибку, если match уже существует.
	Create(ctx context.Context, reward *model.Reward) error

	// FindByMatch ищет правило по ключу match.
	// Возвращает nil, если не найдено.
	FindByMatch(ctx context.Context, match string) (*model.Reward, error)

	// FindAll возвращает все правила (для расчёта начислений).
	FindAll(ctx context.Context) ([]model.Reward, error)
}
