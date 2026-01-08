package dto

type CreateOrderReq struct {
	OrderNumber string `json:"order"`
}

type ItemReq struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type OrderResponse struct {
	Order      string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

type AccrualOrderRequest struct {
	Order string         `json:"order"`
	Goods []OrderItemReq `json:"goods"`
}

type OrderItemReq struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type OrderStatusResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}
