package dto

import "github.com/google/uuid"

type RegisterReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisterRes struct {
	ID    uuid.UUID `json:"user_id"`
	Login string    `json:"login"`
}

type LoginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
