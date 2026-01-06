package orders

import "errors"

var (
	ErrOrderExists = errors.New("заказ уже принят в обработку")
	ErrInvalidData = errors.New("некорректные данные заказа")

	ErrOrderExistsSameUser  = errors.New("order already exists for same user")
	ErrOrderExistsOtherUser = errors.New("order already exists for other user")
	ErrOrderAlreadyHasUser  = errors.New("order already has user")
)
