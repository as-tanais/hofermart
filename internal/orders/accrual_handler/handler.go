package accrual

import (
	"encoding/json"
	"net/http"

	"github.com/as-tanais/hofermart/internal/orders/dto"
	"github.com/as-tanais/hofermart/internal/orders/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type OrderHandler struct {
	service *service.Service
	log     *zap.Logger
}

func NewAccrualHandler(service *service.Service, log *zap.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		log:     log,
	}
}

func (h *OrderHandler) RegisterOrderWithGoods(w http.ResponseWriter, r *http.Request) {
	var req dto.AccrualOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid JSON format", zap.Error(err))
		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}

	if req.Order == "" {
		http.Error(w, "номер заказа обязателен", http.StatusBadRequest)
		return
	}

	if len(req.Goods) == 0 {
		http.Error(w, "список товаров не может быть пустым", http.StatusBadRequest)
		return
	}

	for _, item := range req.Goods {
		if item.Description == "" || item.Price <= 0 {
			http.Error(w, "некорректные данные товара", http.StatusBadRequest)
			return
		}
	}

	err := h.service.RegisterOrderWithGoods(r.Context(), &req)
	if err != nil {
		h.log.Error("Failed to register order with goods",
			zap.String("order", req.Order),
			zap.Error(err))

		switch err.Error() {
		case "invalid order number":
			http.Error(w, "неверный номер заказа", http.StatusBadRequest)
		case "order already processed":
			http.Error(w, "заказ уже обработан", http.StatusConflict)
		default:
			http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
	h.log.Info("Order registered from accrual successfully",
		zap.String("order", req.Order))
}

func (h *OrderHandler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderNumber := chi.URLParam(r, "number")

	if orderNumber == "" {
		http.Error(w, "номер заказа обязателен", http.StatusBadRequest)
		return
	}

	// Получаем статус заказа
	order, err := h.service.GetOrderStatus(r.Context(), orderNumber)
	if err != nil {
		h.log.Error("Failed to get order status",
			zap.String("order", orderNumber),
			zap.Error(err))
		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	if order == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	statusMap := map[string]string{
		"NEW":        "REGISTERED",
		"REGISTERED": "PROCESSING",
		"PROCESSING": "PROCESSING",
		"PROCESSED":  "PROCESSED",
		"INVALID":    "INVALID",
	}

	apiStatus := statusMap[order.Status]
	if apiStatus == "" {
		apiStatus = "INVALID"
	}

	response := dto.OrderStatusResponse{
		Order:   order.OrderNumber,
		Status:  apiStatus,
		Accrual: order.Accrual,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode response",
			zap.String("order", orderNumber),
			zap.Error(err))
	}
}
