package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/as-tanais/hofermart/internal/middleware"
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
	// 1. Проверяем аутентификацию
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.logger.Warn("User not authenticated")
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	// 2. Читаем номер заказа как plain text
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Warn("Failed to read request body",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest) // 400
		return
	}
	defer r.Body.Close()

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		h.logger.Warn("Empty order number",
			zap.String("userID", userID.String()))
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest) // 400
		return
	}

	h.logger.Info("Registering order",
		zap.String("userID", userID.String()),
		zap.String("order", orderNumber))

	// 3. Валидируем номер заказа (алгоритм Луна)
	if !orders.IsValidLuhn(orderNumber) {
		h.logger.Warn("Invalid order number (Luhn check failed)",
			zap.String("userID", userID.String()),
			zap.String("order", orderNumber))
		http.Error(w, "Неверный формат номера заказа", http.StatusUnprocessableEntity) // 422
		return
	}

	// 4. Создаем DTO
	req := dto.CreateOrderReq{
		OrderNumber: orderNumber,
	}

	// 5. Вызываем сервис
	err = h.service.RegisterOrder(r.Context(), userID, &req)
	if err != nil {
		h.logger.Warn("RegisterOrder service error",
			zap.String("userID", userID.String()),
			zap.String("order", orderNumber),
			zap.Error(err))

		// ТОЧНО по спецификации:
		switch {
		case errors.Is(err, orders.ErrInvalidData):
			http.Error(w, "Неверный формат номера заказа", http.StatusUnprocessableEntity) // 422

		case errors.Is(err, orders.ErrOrderExistsSameUser):
			h.logger.Info("Order already registered by same user",
				zap.String("userID", userID.String()),
				zap.String("order", orderNumber))
			w.WriteHeader(http.StatusOK) // 200 - номер заказа уже был загружен этим пользователем

		case errors.Is(err, orders.ErrOrderExistsOtherUser):
			h.logger.Warn("Order already registered by other user",
				zap.String("userID", userID.String()),
				zap.String("order", orderNumber))
			http.Error(w, "Номер заказа уже был загружен другим пользователем", http.StatusConflict) // 409

		default:
			h.logger.Error("Internal server error in RegisterOrder",
				zap.String("userID", userID.String()),
				zap.String("order", orderNumber),
				zap.Error(err))
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError) // 500
		}
		return
	}

	// 6. Успех - новый номер заказа принят
	h.logger.Info("Order accepted for processing",
		zap.String("userID", userID.String()),
		zap.String("order", orderNumber))
	w.WriteHeader(http.StatusAccepted) // 202 - новый номер заказа принят в обработку
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderNumber := chi.URLParam(r, "number")
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

	resp := dto.OrderResponse{
		Order:   order.OrderNumber,
		Status:  apiStatus,
		Accrual: order.Accrual,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetUserOrders - GET /api/user/orders
func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	// 1. Проверяем аутентификацию
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	// 2. Получаем заказы пользователя
	orders, err := h.service.GetUserOrders(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user orders",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// 3. Конвертируем в API формат
	responses := make([]dto.OrderResponse, len(orders))
	for i, order := range orders {
		// Маппинг внутренних статусов на API статусы
		statusMap := map[string]string{
			"NEW":        "NEW",
			"REGISTERED": "PROCESSING",
			"PROCESSING": "PROCESSING",
			"PROCESSED":  "PROCESSED",
			"INVALID":    "INVALID",
		}

		apiStatus := statusMap[order.Status]
		if apiStatus == "" {
			apiStatus = "INVALID"
		}

		// Форматируем дату в RFC3339
		uploadedAt := ""
		if !order.CreatedAt.IsZero() {
			uploadedAt = order.CreatedAt.Format(time.RFC3339)
		}

		responses[i] = dto.OrderResponse{
			Order:      order.OrderNumber,
			Status:     apiStatus,
			Accrual:    order.Accrual,
			UploadedAt: uploadedAt,
		}
	}

	// 4. Возвращаем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(responses); err != nil {
		h.logger.Error("Failed to encode orders response",
			zap.String("userID", userID.String()),
			zap.Error(err))
	}
}
