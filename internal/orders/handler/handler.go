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

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.logger.Warn("User not authenticated")
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Warn("Failed to read request body",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		h.logger.Warn("Empty order number",
			zap.String("userID", userID.String()))
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	h.logger.Info("Registering order",
		zap.String("userID", userID.String()),
		zap.String("order", orderNumber))

	if !orders.IsValidLuhn(orderNumber) {
		h.logger.Warn("Invalid order number (Luhn check failed)",
			zap.String("userID", userID.String()),
			zap.String("order", orderNumber))
		http.Error(w, "Неверный формат номера заказа", http.StatusUnprocessableEntity)
		return
	}

	req := dto.CreateOrderReq{
		OrderNumber: orderNumber,
	}

	err = h.service.RegisterOrder(r.Context(), userID, &req)
	if err != nil {
		h.logger.Warn("RegisterOrder service error",
			zap.String("userID", userID.String()),
			zap.String("order", orderNumber),
			zap.Error(err))

		switch {
		case errors.Is(err, orders.ErrInvalidData):
			http.Error(w, "Неверный формат номера заказа", http.StatusUnprocessableEntity)

		case errors.Is(err, orders.ErrOrderExistsSameUser):
			h.logger.Info("Order already registered by same user",
				zap.String("userID", userID.String()),
				zap.String("order", orderNumber))
			w.WriteHeader(http.StatusOK)

		case errors.Is(err, orders.ErrOrderExistsOtherUser):
			h.logger.Warn("Order already registered by other user",
				zap.String("userID", userID.String()),
				zap.String("order", orderNumber))
			http.Error(w, "Номер заказа уже был загружен другим пользователем", http.StatusConflict)

		default:
			h.logger.Error("Internal server error in RegisterOrder",
				zap.String("userID", userID.String()),
				zap.String("order", orderNumber),
				zap.Error(err))
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Info("Order accepted for processing",
		zap.String("userID", userID.String()),
		zap.String("order", orderNumber))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Пользователь не аутентифицирован", http.StatusUnauthorized)
		return
	}

	orders, err := h.service.GetUserOrders(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user orders",
			zap.String("userID", userID.String()),
			zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	responses := make([]dto.OrderResponse, len(orders))
	for i, order := range orders {

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(responses); err != nil {
		h.logger.Error("Failed to encode orders response",
			zap.String("userID", userID.String()),
			zap.Error(err))
	}
}
