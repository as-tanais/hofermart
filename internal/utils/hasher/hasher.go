package hasher

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Hasher struct {
	cost int
}

func NewHasher(cost int) *Hasher {
	return &Hasher{
		cost: cost,
	}
}

func (h *Hasher) HashPassword(password string) (string, error) {

	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("не получилось захэшировать пароль: %w", err)
	}

	return string(hash), nil
}

func (h *Hasher) Compare(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
