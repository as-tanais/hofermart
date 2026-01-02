package orders

import "errors"

var (
	ErrOrderExists = errors.New("заказ уже принят в обработку")
	ErrInvalidData = errors.New("некорректные данные заказа")
)
