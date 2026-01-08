package storage

import (
	"context"
	"errors"
	"fmt"

	usrerr "github.com/as-tanais/hofermart/internal/user"
	"github.com/as-tanais/hofermart/internal/user/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userStorage struct {
	db *pgxpool.Pool
}

func NewUserStorage(db *pgxpool.Pool) UserStorage {
	return &userStorage{
		db: db,
	}
}

func (s *userStorage) Create(ctx context.Context, user *model.User) (*model.User, error) {

	const insertQuery = `
        INSERT INTO users (login, password_hash)
        VALUES ($1, $2)
        ON CONFLICT (login) DO NOTHING
        RETURNING id;
    `

	err := s.db.QueryRow(ctx, insertQuery, user.Login, user.Password).Scan(&user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, usrerr.ErrLoginExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *userStorage) FindByLogin(ctx context.Context, login string) (*model.User, error) {
	const query = `
		SELECT id, login, password_hash
		FROM users
		WHERE login = $1;
	`

	user := &model.User{}
	err := s.db.QueryRow(ctx, query, login).Scan(&user.ID, &user.Login, &user.Password)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

func (s *userStorage) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return nil, nil
}
