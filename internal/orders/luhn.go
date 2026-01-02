// Package orders содержит вспомогательные функции для заказов.
package orders

import (
	"strconv"
	"unicode"
)

// IsValidLuhn проверяет, проходит ли строка проверку по алгоритму Луна.
// Строка должна содержать только цифры.
func IsValidLuhn(s string) bool {
	if s == "" {
		return false
	}

	// Проверяем, что все символы — цифры
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	n := len(s)
	sum := 0
	double := false

	// Проходим справа налево
	for i := n - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(s[i]))
		if double {
			digit *= 2
			if digit > 9 {
				digit = digit%10 + digit/10
			}
		}
		sum += digit
		double = !double
	}

	return sum%10 == 0
}
