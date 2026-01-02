package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/as-tanais/hofermart/internal/rewards"
	"github.com/as-tanais/hofermart/internal/rewards/dto"
	"github.com/as-tanais/hofermart/internal/rewards/service"

	"go.uber.org/zap"
)

type Handler struct {
	service *service.Service
	logger  *zap.Logger
}

func NewHandler(service *service.Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) CreateReward(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRewardReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateReward(&req); err != nil {
		h.logger.Warn("CreateReward error", zap.Error(err))

		switch {
		case errors.Is(err, rewards.ErrInvalidData):
			http.Error(w, "некорректные данные вознаграждения", http.StatusBadRequest)
		case errors.Is(err, rewards.ErrMatchExists):
			http.Error(w, "ключ поиска уже зарегистрирован", http.StatusConflict)
		default:
			http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
