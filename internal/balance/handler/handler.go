package handler

import (
	"encoding/json"
	"net/http"

	"github.com/as-tanais/hofermart/internal/balance/service"
	"github.com/as-tanais/hofermart/internal/middleware"
	"go.uber.org/zap"
)

type BalanceHandler struct {
	service *service.BalanceService
	log     *zap.Logger
}

func NewBalanceHandler(service *service.BalanceService, log *zap.Logger) *BalanceHandler {
	return &BalanceHandler{
		service: service,
		log:     log,
	}
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	balance, err := h.service.GetBalance(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get user balance",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(balance); err != nil {
		h.log.Error("Failed to encode balance response",
			zap.String("userID", userID.String()),
			zap.Error(err))
	}
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	var withdrawRequest struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}

	if err := json.NewDecoder(r.Body).Decode(&withdrawRequest); err != nil {
		h.log.Warn("Failed to decode withdraw request",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if withdrawRequest.Order == "" || withdrawRequest.Sum <= 0 {
		h.log.Warn("Invalid withdraw request data",
			zap.String("userID", userID.String()),
			zap.String("order", withdrawRequest.Order),
			zap.Float64("sum", withdrawRequest.Sum))
		http.Error(w, "Неверные данные запроса", http.StatusBadRequest)
		return
	}

	err := h.service.Withdraw(r.Context(), userID, withdrawRequest.Order, withdrawRequest.Sum)
	if err != nil {
		switch err {
		case service.ErrInvalidOrderNumber:
			http.Error(w, "Неверный номер заказа", http.StatusUnprocessableEntity)
		case service.ErrInsufficientFunds:
			http.Error(w, "На счету недостаточно средств", http.StatusPaymentRequired)
		case service.ErrOrderAlreadyExists:
			http.Error(w, "Заказ уже был использован для списания", http.StatusUnprocessableEntity)
		default:
			h.log.Error("Failed to process withdrawal",
				zap.String("userID", userID.String()),
				zap.String("order", withdrawRequest.Order),
				zap.Float64("sum", withdrawRequest.Sum),
				zap.Error(err))
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.service.GetUserWithdrawals(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get user withdrawals",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		h.log.Error("Failed to encode withdrawals response",
			zap.String("userID", userID.String()),
			zap.Error(err))
	}
}
