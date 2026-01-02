package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/as-tanais/hofermart/internal/orders"
	"github.com/as-tanais/hofermart/internal/orders/dto"
	"github.com/as-tanais/hofermart/internal/orders/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler struct {
	service *service.Service
	logger  *zap.Logger
}

func NewHandler(service *service.Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterOrder(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.service.RegisterOrder(r.Context(), &req); err != nil {
		h.logger.Warn("RegisterOrder error", zap.Error(err))

		switch {
		case errors.Is(err, orders.ErrInvalidData):
			http.Error(w, "некорректные данные заказа", http.StatusBadRequest)
		case errors.Is(err, orders.ErrOrderExists):
			http.Error(w, "заказ уже принят в обработку", http.StatusConflict)
		default:
			http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderNumber := chi.URLParam(r, "number") // используем chi

	if !orders.IsValidLuhn(orderNumber) {
		http.Error(w, "некорректный номер заказа", http.StatusBadRequest)
		return
	}

	order, err := h.service.GetOrder(r.Context(), orderNumber)
	if err != nil {
		// ...
		return
	}
	if order == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Маппинг статусов
	statusMap := map[string]string{
		"new":        "REGISTERED",
		"processing": "PROCESSING",
		"processed":  "PROCESSED",
		"invalid":    "INVALID",
	}
	apiStatus := statusMap[order.Status]
	if apiStatus == "" {
		apiStatus = "INVALID"
	}

	// Формируем DTO для ответа (без Items!)
	resp := dto.OrderResponse{
		Order:   order.OrderNumber,
		Status:  apiStatus,
		Accrual: order.Accrual,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
