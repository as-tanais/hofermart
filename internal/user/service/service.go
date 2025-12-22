package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	usrerr "github.com/as-tanais/hofermart/internal/user"
	"github.com/as-tanais/hofermart/internal/user/dto"
	"github.com/as-tanais/hofermart/internal/user/model"
	"github.com/as-tanais/hofermart/internal/user/storage"
	"github.com/as-tanais/hofermart/internal/utils/hasher"
	"go.uber.org/zap"
)

type UserService struct {
	storage storage.UserStorage
	hasher  *hasher.Hasher
	logger  *zap.Logger
}

func NewUserService(storage storage.UserStorage, hasher *hasher.Hasher, logger *zap.Logger) *UserService {
	return &UserService{
		storage: storage,
		hasher:  hasher,
		logger:  logger,
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
		s.logger.Error("Failed to hash password",
			zap.String("login", req.Login),
			zap.Error(err),
		)
		return nil, fmt.Errorf("ошибка хеширования")
	}

	newUser := &model.User{Login: req.Login, Password: hash}
	created, err := s.storage.Create(ctx, newUser)
	if err != nil {
		if errors.Is(err, usrerr.ErrLoginExists) {
			return nil, usrerr.ErrLoginExists
		}
		s.logger.Error("Failed to create user in DB",
			zap.String("login", req.Login),
			zap.Error(err),
		)
		return nil, fmt.Errorf("ошибка создания пользователя")
	}

	return &dto.RegisterRes{ID: created.ID, Login: created.Login}, nil
}

func (s *UserService) Login(ctx context.Context, login, password string) (*dto.RegisterRes, error) {
	if login == "" || password == "" {
		return nil, usrerr.ErrInvalidCredentials
	}

	dbUser, err := s.storage.FindByLogin(ctx, login)
	if err != nil {
		return nil, usrerr.ErrInvalidCredentials
	}

	if !s.hasher.Compare(password, dbUser.Password) {
		return nil, usrerr.ErrInvalidCredentials
	}

	r := &dto.RegisterRes{
		ID:    dbUser.ID,
		Login: dbUser.Login,
	}

	return r, nil
}

func (s *UserService) validateLogin(login string) error {
	if login == "" {
		return usrerr.ErrInvalidLogin
	}

	allowed := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !allowed.MatchString(login) {
		return usrerr.ErrInvalidLogin
	}
	return nil
}

// validatePassword проверяет сложность пароля.
func (s *UserService) validatePassword(password string) error {
	if len(password) < 6 {
		return usrerr.ErrInvalidPassword
	}
	return nil
}
