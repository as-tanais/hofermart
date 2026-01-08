package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	balerr "github.com/as-tanais/hofermart/internal/balance"
	"github.com/as-tanais/hofermart/internal/balance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
}

func NewPostgresStorage(db *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) GetBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	query := `
        SELECT 
            COALESCE(SUM(CASE 
                WHEN o.status = 'PROCESSED' THEN o.accrual 
                ELSE 0 
            END), 0) - 
            COALESCE(SUM(w.sum), 0) as current,
            COALESCE(SUM(w.sum), 0) as withdrawn
        FROM users u
        LEFT JOIN orders o ON o.user_id = u.id
        LEFT JOIN withdrawals w ON w.user_id = u.id
        WHERE u.id = $1
        GROUP BY u.id
    `

	var balance model.Balance
	balance.UserID = userID

	err := s.db.QueryRow(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		// Проверяем, если это ошибка "no rows"
		if errors.Is(err, pgx.ErrNoRows) {
			// Возвращаем нулевой баланс вместо ошибки
			return &model.Balance{
				UserID:    userID,
				Current:   0,
				Withdrawn: 0,
			}, nil
		}
		return nil, fmt.Errorf("%w: failed to get balance: %v", balerr.ErrDatabaseError, err)
	}

	return &balance, nil
}

func (s *PostgresStorage) CreateWithdrawal(ctx context.Context, userID uuid.UUID, orderNumber string, sum float64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", balerr.ErrDatabaseError, err)
	}
	defer tx.Rollback(ctx)

	var currentBalance float64
	balanceQuery := `
        SELECT 
            COALESCE(SUM(CASE 
                WHEN o.status = 'PROCESSED' THEN o.accrual 
                ELSE 0 
            END), 0) - 
            COALESCE(SUM(w.sum), 0) as current
        FROM users u
        LEFT JOIN orders o ON o.user_id = u.id
        LEFT JOIN withdrawals w ON w.user_id = u.id
        WHERE u.id = $1
        GROUP BY u.id
    `

	err = tx.QueryRow(ctx, balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		// Если нет записей, баланс = 0
		if errors.Is(err, pgx.ErrNoRows) {
			currentBalance = 0
		} else {
			return fmt.Errorf("%w: failed to get current balance: %v", balerr.ErrDatabaseError, err)
		}
	}

	// Проверяем достаточно ли средств
	if currentBalance < sum {
		return fmt.Errorf("%w: current=%.2f, requested=%.2f", balerr.ErrInsufficientFunds, currentBalance, sum)
	}

	// Проверяем уникальность номера заказа
	var exists bool
	checkQuery := `
        SELECT EXISTS(
            SELECT 1 FROM withdrawals WHERE order_number = $1
            UNION ALL
            SELECT 1 FROM orders WHERE order_number = $1
        )
    `
	err = tx.QueryRow(ctx, checkQuery, orderNumber).Scan(&exists)
	if err != nil {
		return fmt.Errorf("%w: failed to check order uniqueness: %v", balerr.ErrDatabaseError, err)
	}

	if exists {
		return fmt.Errorf("%w: %s", balerr.ErrOrderAlreadyExists, orderNumber)
	}

	// Создаем вывод средств
	insertQuery := `
        INSERT INTO withdrawals (id, user_id, order_number, sum, processed_at)
        VALUES ($1, $2, $3, $4, $5)
    `

	withdrawalID := uuid.New()
	_, err = tx.Exec(ctx, insertQuery,
		withdrawalID,
		userID,
		orderNumber,
		sum,
		time.Now(),
	)

	if err != nil {
		// Обрабатываем ошибку дублирования
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s", balerr.ErrOrderAlreadyExists, orderNumber)
		}
		return fmt.Errorf("%w: failed to create withdrawal: %v", balerr.ErrDatabaseError, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", balerr.ErrDatabaseError, err)
	}

	return nil
}

func (s *PostgresStorage) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.Withdrawal, error) {
	query := `
        SELECT id, user_id, order_number, sum, processed_at
        FROM withdrawals 
        WHERE user_id = $1
        ORDER BY processed_at DESC
    `

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query withdrawals: %v", balerr.ErrDatabaseError, err)
	}
	defer rows.Close()

	var withdrawals []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		var processedAt time.Time

		err := rows.Scan(&w.ID, &w.UserID, &w.Order, &w.Sum, &processedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan withdrawal: %v", balerr.ErrDatabaseError, err)
		}

		w.ProcessedAt = processedAt.Format(time.RFC3339)
		withdrawals = append(withdrawals, w)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: rows iteration error: %v", balerr.ErrDatabaseError, err)
	}

	return withdrawals, nil
}

func (s *PostgresStorage) CheckOrderExists(ctx context.Context, orderNumber string) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 FROM withdrawals WHERE order_number = $1
            UNION ALL
            SELECT 1 FROM orders WHERE number = $1
        )
    `

	var exists bool
	err := s.db.QueryRow(ctx, query, orderNumber).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%w: failed to check order existence: %v", balerr.ErrDatabaseError, err)
	}

	return exists, nil
}
