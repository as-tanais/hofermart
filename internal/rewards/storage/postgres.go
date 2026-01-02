package storage

import (
	"context"
	"errors"
	"fmt"

	rewerr "github.com/as-tanais/hofermart/internal/rewards"
	"github.com/as-tanais/hofermart/internal/rewards/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStorage struct {
	db *pgxpool.Pool
}

func NewPostgresStorage(db *pgxpool.Pool) Repository {
	return &postgresStorage{db: db}
}

func (s *postgresStorage) Create(ctx context.Context, r *model.Reward) error {
	const query = `
        INSERT INTO goods_rewards (match, reward, reward_type)
        VALUES ($1, $2, $3)
    `

	_, err := s.db.Exec(ctx, query, r.Match, r.Reward, r.RewardType)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return rewerr.ErrMatchExists
		}
		return fmt.Errorf("failed to create reward: %w", err)
	}
	return nil
}

func (s *postgresStorage) FindByMatch(ctx context.Context, match string) (*model.Reward, error) {
	const query = `
        SELECT id, match, reward, reward_type
        FROM goods_rewards
        WHERE match = $1
    `

	reward := &model.Reward{}
	err := s.db.QueryRow(ctx, query, match).Scan(
		&reward.ID, &reward.Match, &reward.Reward, &reward.RewardType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // не найдено — не ошибка
		}
		return nil, fmt.Errorf("failed to find reward by match: %w", err)
	}
	return reward, nil
}

func (s *postgresStorage) FindAll(ctx context.Context) ([]model.Reward, error) {
	const query = `
        SELECT id, match, reward, reward_type
        FROM goods_rewards
    `

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all rewards: %w", err)
	}
	defer rows.Close()

	var rewards []model.Reward
	for rows.Next() {
		var r model.Reward
		err := rows.Scan(&r.ID, &r.Match, &r.Reward, &r.RewardType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reward: %w", err)
		}
		rewards = append(rewards, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return rewards, nil
}
