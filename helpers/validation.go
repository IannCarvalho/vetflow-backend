package helpers

import (
	"strings"
	"unicode"
)

// onlyDigits returns a string containing just the digit runes of s, dropping
// any mask characters (dots, dashes, parentheses, spaces, etc.).
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// allSameDigit reports whether every rune in digits is the same character.
// It assumes digits is non-empty.
func allSameDigit(digits string) bool {
	for i := 1; i < len(digits); i++ {
		if digits[i] != digits[0] {
			return false
		}
	}
	return true
}

// cpfCheckDigit computes a single CPF verification digit for the first n digits
// using the official weighting scheme (weights run from n+1 down to 2).
func cpfCheckDigit(digits string, n int) int {
	sum := 0
	weight := n + 1
	for i := range n {
		sum += int(digits[i]-'0') * weight
		weight--
	}
	rem := 11 - (sum % 11)
	if rem >= 10 {
		return 0
	}
	return rem
}

// IsValidCPF reports whether cpf is a valid Brazilian CPF. It accepts values
// with or without the usual mask (e.g. "529.982.247-25" or "52998224725").
func IsValidCPF(cpf string) bool {
	digits := onlyDigits(cpf)
	if len(digits) != 11 {
		return false
	}
	if allSameDigit(digits) {
		return false
	}
	dv1 := cpfCheckDigit(digits, 9)
	if dv1 != int(digits[9]-'0') {
		return false
	}
	dv2 := cpfCheckDigit(digits, 10)
	return dv2 == int(digits[10]-'0')
}

// IsValidBRPhone reports whether phone is a valid Brazilian phone number,
// accepting 10 digits (landline) or 11 digits (mobile), with or without a mask.
func IsValidBRPhone(phone string) bool {
	n := len(onlyDigits(phone))
	return n == 10 || n == 11
}

// IsValidCEP reports whether cep is a valid Brazilian postal code (exactly 8
// digits), with or without a mask (e.g. "01234-567" or "01234567").
func IsValidCEP(cep string) bool {
	return len(onlyDigits(cep)) == 8
}
