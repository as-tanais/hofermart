package user

import "errors"

var (
	ErrInvalidLogin = errors.New("некорректный логин")

	ErrInvalidPassword = errors.New("некорректный пароль")

	ErrLoginExists = errors.New("логин уже занят")

	ErrUserNotFound = errors.New("пользователь не найден")

	ErrInvalidCredentials = errors.New("неверный логин или пароль")
)
