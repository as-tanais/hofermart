package model

import "github.com/google/uuid"

// Reward информации о вознаграждении за товар
type Reward struct {
	ID         uuid.UUID `json:"id"`
	Match      string    `json:"match"`
	Reward     float64   `json:"reward"`
	RewardType string    `json:"reward_type"`
}
