package storage

import (
	"context"

	"github.com/as-tanais/hofermart/internal/user/model"
	"github.com/google/uuid"
)

type UserStorage interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	FindByLogin(context.Context, string) (*model.User, error)
	FindByID(context.Context, uuid.UUID) (*model.User, error)
}
