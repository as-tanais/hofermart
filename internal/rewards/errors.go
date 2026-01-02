package rewards

import "errors"

var (
	ErrMatchExists = errors.New("ключ поиска уже зарегистрирован")
	ErrInvalidData = errors.New("некорректные данные вознаграждения")
)
