package orders

import (
	"strconv"
	"unicode"
)

func IsValidLuhn(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	n := len(s)
	sum := 0
	double := false

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
