package dto

type CreateOrderReq struct {
	OrderNumber string    `json:"order"`
	Goods       []ItemReq `json:"goods"`
}

type ItemReq struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type OrderResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}
