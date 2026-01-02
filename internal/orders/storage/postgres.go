package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	orderr "github.com/as-tanais/hofermart/internal/orders"
	"github.com/as-tanais/hofermart/internal/orders/model"
	"github.com/google/uuid"
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

func (s *postgresStorage) OrderExists(ctx context.Context, orderNumber string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM orders WHERE order_number = $1)", orderNumber).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check order existence: %w", err)
	}
	return exists, nil
}

func (s *postgresStorage) SaveOrder(ctx context.Context, order *model.Order) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Сохраняем заказ
	var orderID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (order_number, status)
		VALUES ($1, 'new')
		RETURNING id;
	`, order.OrderNumber).Scan(&orderID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return orderr.ErrOrderExists
		}
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Сохраняем товары
	for _, item := range order.Items {
		_, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, description, price)
			VALUES ($1, $2, $3);
		`, orderID, item.Description, item.Price)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *postgresStorage) GetOrderByNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	const query = `
		SELECT id, order_number, status, accrual
		FROM orders
		WHERE order_number = $1
	`

	order := &model.Order{}
	var accrual sql.NullFloat64

	err := s.db.QueryRow(ctx, query, orderNumber).Scan(
		&order.ID, &order.OrderNumber, &order.Status, &accrual,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if accrual.Valid {
		order.Accrual = &accrual.Float64
	}

	return order, nil
}
