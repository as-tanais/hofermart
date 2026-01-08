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
	query := `
		INSERT INTO orders (id, user_id, order_number, status)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.db.Exec(ctx, query,
		order.ID,
		order.UserID,
		order.OrderNumber,
		order.Status,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return orderr.ErrOrderExists
		}
		return fmt.Errorf("failed to insert order: %w", err)
	}

	return nil
}

func (s *postgresStorage) GetOrderByNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	return s.FindByNumber(ctx, orderNumber)
}

func (s *postgresStorage) SaveOrderItems(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, description, price)
			VALUES ($1, $2, $3, $4)
		`, item.ID, orderID, item.Description, item.Price)

		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *postgresStorage) Update(ctx context.Context, order *model.Order) error {
	query := `
		UPDATE orders 
		SET 
			status = COALESCE($2, status),
			accrual = COALESCE($3, accrual),
			user_id = COALESCE($4, user_id),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := s.db.Exec(ctx, query,
		order.ID,
		order.Status,
		order.Accrual,
		order.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update order %s: %w", order.ID, err)
	}

	return nil
}

func (s *postgresStorage) GetOrdersByStatus(ctx context.Context, status string, limit int) ([]model.Order, error) {
	query := `
		SELECT id, user_id, order_number, status, accrual, created_at
		FROM orders
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders by status: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		var userIDStr *string
		var accrual sql.NullFloat64

		err := rows.Scan(
			&order.ID,
			&userIDStr,
			&order.OrderNumber,
			&order.Status,
			&accrual,
			&order.CreatedAt,
		)
		if err != nil {
			return orders, fmt.Errorf("failed to scan order: %w", err)
		}

		if userIDStr != nil {
			order.UserID, err = uuid.Parse(*userIDStr)
			if err != nil {
				return orders, fmt.Errorf("invalid user ID format: %w", err)
			}
		}

		if accrual.Valid {
			order.Accrual = &accrual.Float64
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return orders, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}

func (s *postgresStorage) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error) {
	query := `
		SELECT id, order_id, description, price
		FROM order_items
		WHERE order_id = $1
	`

	rows, err := s.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		var orderIDStr string

		err := rows.Scan(
			&item.ID,
			&orderIDStr,
			&item.Description,
			&item.Price,
		)
		if err != nil {
			return items, fmt.Errorf("failed to scan order item: %w", err)
		}

		item.OrderID, err = uuid.Parse(orderIDStr)
		if err != nil {
			return items, fmt.Errorf("invalid order ID format: %w", err)
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return items, fmt.Errorf("rows error: %w", err)
	}

	return items, nil
}

func (s *postgresStorage) GetOrderForRegistration(ctx context.Context, orderNumber string) (*model.Order, error) {
	order, err := s.FindByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	// Фильтрация по статусу остается здесь
	if order != nil && order.Status != "NEW" {
		return nil, nil // Или можно вернуть ошибку
	}

	return order, nil
}

func (s *postgresStorage) UpdateOrderToRegistered(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE orders 
		SET status = 'REGISTERED'
		WHERE id = $1 AND status = 'NEW'
	`, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Сохраняем товары
	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, description, price)
			VALUES ($1, $2, $3, $4)
		`, item.ID, orderID, item.Description, item.Price)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *postgresStorage) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.Order, error) {
	query := `
		SELECT id, order_number, status, accrual, created_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		var accrual sql.NullFloat64

		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.Status,
			&accrual,
			&order.CreatedAt,
		)
		if err != nil {
			return orders, fmt.Errorf("failed to scan order: %w", err)
		}

		order.UserID = userID
		if accrual.Valid {
			order.Accrual = &accrual.Float64
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return orders, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}

func (s *postgresStorage) UpdateOrderUser(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE orders 
		SET user_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND user_id IS NULL
	`

	result, err := s.db.Exec(ctx, query, userID, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order user: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("order already has user")
	}

	return nil
}

func (s *postgresStorage) UpdateOrderUserAndStatus(ctx context.Context, orderID, userID uuid.UUID, status string) error {

	query := `
		UPDATE orders 
		SET user_id = $1, status = $2
		WHERE id = $3 AND (user_id IS NULL OR user_id = '00000000-0000-0000-0000-000000000000')
	`

	result, err := s.db.Exec(ctx, query, userID, status, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {

		var currentUserID uuid.UUID
		var currentStatus string
		checkQuery := `SELECT user_id, status FROM orders WHERE id = $1`
		err = s.db.QueryRow(ctx, checkQuery, orderID).Scan(&currentUserID, &currentStatus)
		if err == nil {
			return fmt.Errorf("order already has user %s and status %s",
				currentUserID.String(), currentStatus)
		}
		return fmt.Errorf("order not found")
	}

	return nil
}

// FindByNumber заменяет GetOrderByNumber и GetOrderForRegistration
func (s *postgresStorage) FindByNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	const query = `
		SELECT id, user_id, order_number, status, accrual, created_at
		FROM orders
		WHERE order_number = $1
	`

	order := &model.Order{}
	var userIDStr *string
	var accrual sql.NullFloat64

	err := s.db.QueryRow(ctx, query, orderNumber).Scan(
		&order.ID,
		&userIDStr,
		&order.OrderNumber,
		&order.Status,
		&accrual,
		&order.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Заказ не найден
		}
		return nil, fmt.Errorf("failed to find order by number: %w", err)
	}

	// Конвертируем userID если он есть
	if userIDStr != nil {
		order.UserID, err = uuid.Parse(*userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID format: %w", err)
		}
	}

	if accrual.Valid {
		order.Accrual = &accrual.Float64
	}

	return order, nil
}
