// internal/balance/storage/postgres.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/as-tanais/hofermart/internal/balance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error)
	CreateWithdrawal(ctx context.Context, userID uuid.UUID, orderNumber string, sum float64) error
	GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.Withdrawal, error)
	CheckOrderExists(ctx context.Context, orderNumber string) (bool, error)
}

type PostgresStorage struct {
	db *pgxpool.Pool
}

func NewPostgresStorage(db *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{db: db}
}

// GetBalance возвращает баланс пользователя (рассчитывается на лету)
func (s *PostgresStorage) GetBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	query := `
        WITH accrued AS (
            SELECT COALESCE(SUM(accrual), 0) as total
            FROM orders 
            WHERE user_id = $1 
              AND status = 'PROCESSED'
        ),
        withdrawn AS (
            SELECT COALESCE(SUM(sum), 0) as total
            FROM withdrawals 
            WHERE user_id = $1
        )
        SELECT 
            (SELECT total FROM accrued) - (SELECT total FROM withdrawn) as current,
            (SELECT total FROM withdrawn) as withdrawn
    `

	var balance model.Balance
	balance.UserID = userID

	err := s.db.QueryRow(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance for user %s: %w", userID, err)
	}

	return &balance, nil
}

// CreateWithdrawal создает списание с проверкой баланса
func (s *PostgresStorage) CreateWithdrawal(ctx context.Context, userID uuid.UUID, orderNumber string, sum float64) error {
	// Начинаем транзакцию
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Проверяем баланс пользователя с блокировкой
	var currentBalance float64
	balanceQuery := `
        WITH accrued AS (
            SELECT COALESCE(SUM(accrual), 0) as total
            FROM orders 
            WHERE user_id = $1 
              AND status = 'PROCESSED'
              FOR UPDATE
        ),
        withdrawn AS (
            SELECT COALESCE(SUM(sum), 0) as total
            FROM withdrawals 
            WHERE user_id = $1
              FOR UPDATE
        )
        SELECT (SELECT total FROM accrued) - (SELECT total FROM withdrawn) as current
    `

	err = tx.QueryRow(ctx, balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get current balance for user %s: %w", userID, err)
	}

	// 2. Проверяем достаточно ли средств
	if currentBalance < sum {
		return fmt.Errorf("insufficient funds: current=%.2f, requested=%.2f", currentBalance, sum)
	}

	// 3. Проверяем уникальность номера заказа
	// Проверяем как в withdrawals, так и в orders (чтобы один номер не использовался дважды)
	var exists bool
	checkQuery := `
        SELECT EXISTS(
            SELECT 1 FROM withdrawals WHERE order_number = $1
            UNION ALL
            SELECT 1 FROM orders WHERE number = $1
        )
    `
	err = tx.QueryRow(ctx, checkQuery, orderNumber).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check order uniqueness: %w", err)
	}

	if exists {
		return fmt.Errorf("order number %s already exists", orderNumber)
	}

	// 4. Создаем запись о списании с UUID
	insertQuery := `
        INSERT INTO withdrawals (id, user_id, order_number, sum, processed_at)
        VALUES ($1, $2, $3, $4, $5)
    `

	withdrawalID := uuid.New()
	_, err = tx.Exec(ctx, insertQuery,
		withdrawalID, // $1 - UUID записи
		userID,       // $2 - UUID пользователя
		orderNumber,  // $3 - номер заказа
		sum,          // $4 - сумма
		time.Now(),   // $5 - время списания
	)

	if err != nil {
		// Проверяем нарушение уникальности (на случай race condition)
		if isUniqueViolation(err) {
			return fmt.Errorf("order number %s already exists", orderNumber)
		}
		return fmt.Errorf("failed to create withdrawal with ID %s: %w", withdrawalID, err)
	}

	// 5. Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// isUniqueViolation проверяет, является ли ошибка нарушением уникальности в pgx/v5
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok {
		// SQLState 23505 - unique_violation
		// SQLState 23503 - foreign_key_violation (если нужно проверять)
		return pgErr.Code == "23505"
	}
	return false
}

// GetUserWithdrawals возвращает историю списаний пользователя
func (s *PostgresStorage) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.Withdrawal, error) {
	query := `
        SELECT id, user_id, order_number, sum, processed_at
        FROM withdrawals 
        WHERE user_id = $1
        ORDER BY processed_at DESC
    `

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query withdrawals for user %s: %w", userID, err)
	}
	defer rows.Close()

	var withdrawals []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		var processedAt time.Time

		err := rows.Scan(&w.ID, &w.UserID, &w.Order, &w.Sum, &processedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan withdrawal: %w", err)
		}

		w.ProcessedAt = processedAt.Format(time.RFC3339Nano)
		withdrawals = append(withdrawals, w)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return withdrawals, nil
}

// CheckOrderExists проверяет, существует ли уже заказ
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
		return false, fmt.Errorf("failed to check existence of order %s: %w", orderNumber, err)
	}

	return exists, nil
}
