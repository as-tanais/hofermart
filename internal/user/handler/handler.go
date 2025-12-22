package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/as-tanais/hofermart/internal/auth"
	"github.com/as-tanais/hofermart/internal/user"
	"github.com/as-tanais/hofermart/internal/user/dto"
	"github.com/as-tanais/hofermart/internal/user/service"
	"go.uber.org/zap"
)

type Handler struct {
	service    *service.UserService
	jwtManager *auth.JWTManager
	logger     *zap.Logger
}

func NewHandler(service *service.UserService, jwtManager *auth.JWTManager, logger *zap.Logger) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}

	createdUser, err := h.service.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrInvalidLogin), errors.Is(err, user.ErrInvalidPassword):
			http.Error(w, "некорректный логин или пароль", http.StatusBadRequest)
		case errors.Is(err, user.ErrLoginExists):
			http.Error(w, "логин уже занят", http.StatusConflict)
		case errors.Is(err, user.ErrInvalidCredentials):
			http.Error(w, "неверный логин или пароль", http.StatusUnauthorized)
		default:
			http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	token, err := h.jwtManager.GenerateToken(createdUser.ID)
	if err != nil {
		h.logger.Error("Не удалось сгенерировать JWT", zap.Error(err))
		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.jwtManager.TokenDuration().Seconds()),
	})

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "логин и пароль обязательны", http.StatusBadRequest)
		return
	}

	u, err := h.service.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		h.logger.Warn("Ошибка входа",
			zap.String("login", req.Login),
			zap.Error(err),
		)

		if errors.Is(err, user.ErrInvalidCredentials) {
			http.Error(w, "неверный логин или пароль", http.StatusUnauthorized)
			return
		}

		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtManager.GenerateToken(u.ID)
	if err != nil {
		h.logger.Error("Не удалось сгенерировать JWT", zap.Error(err))
		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.jwtManager.TokenDuration().Seconds()),
	})

	w.WriteHeader(http.StatusOK)
}
