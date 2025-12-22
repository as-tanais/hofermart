package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/as-tanais/hofermart/internal/user"
	"github.com/as-tanais/hofermart/internal/user/dto"
	"github.com/as-tanais/hofermart/internal/user/model"
	"github.com/as-tanais/hofermart/internal/user/storage"
	"github.com/as-tanais/hofermart/internal/utils/hasher"
)

type UserService struct {
	storage storage.UserStorage
	hasher  *hasher.Hasher
}

func NewUserService(storage storage.UserStorage, hasher *hasher.Hasher) *UserService {
	return &UserService{
		storage: storage,
		hasher:  hasher,
	}
}

func (s *UserService) Register(ctx context.Context, req dto.RegisterReq) (*dto.RegisterRes, error) {

	if err := s.validateLogin(req.Login); err != nil {
		return nil, err
	}
	if err := s.validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := s.hasher.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	fmt.Println("Proverili login i pass i zashirovali pass")

	newUser := &model.User{Login: req.Login, Password: hash}
	created, err := s.storage.Create(ctx, newUser)
	if err != nil {

		if err.Error() == `login "`+req.Login+`" already exists` {
			return nil, user.ErrLoginExists
		}
		return nil, err
	}

	r := &dto.RegisterRes{
		ID:    created.ID,
		Login: created.Login,
	}

	return r, nil
}

func (s *UserService) Login(ctx context.Context, login, password string) (*dto.RegisterRes, error) {
	if login == "" || password == "" {
		return nil, user.ErrInvalidCredentials
	}

	dbUser, err := s.storage.FindByLogin(ctx, login)
	if err != nil {
		return nil, user.ErrInvalidCredentials
	}

	if !s.hasher.Compare(password, dbUser.Password) {
		return nil, user.ErrInvalidCredentials
	}

	r := &dto.RegisterRes{
		ID:    dbUser.ID,
		Login: dbUser.Login,
	}

	return r, nil
}

func (s *UserService) validateLogin(login string) error {
	if login == "" {
		return user.ErrInvalidLogin
	}

	allowed := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !allowed.MatchString(login) {
		return user.ErrInvalidLogin
	}
	return nil
}

// validatePassword проверяет сложность пароля.
func (s *UserService) validatePassword(password string) error {
	if len(password) < 6 {
		return user.ErrInvalidPassword
	}
	return nil
}
