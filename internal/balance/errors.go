package balance

import "errors"

var (
	ErrInsufficientFunds = errors.New("insufficient funds")

	ErrInvalidOrderNumber = errors.New("invalid order number")

	ErrOrderAlreadyExists = errors.New("order already exists")

	ErrNegativeAmount = errors.New("amount must be positive")

	ErrBalanceNotFound = errors.New("balance not found")

	ErrWithdrawalNotFound = errors.New("withdrawal not found")

	ErrDatabaseError = errors.New("database error")
)
