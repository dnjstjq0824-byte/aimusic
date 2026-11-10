package domain

import (
	"fmt"
	"strings"
)

// biznoWeights are the 국세청 verification multipliers for digits 1..8.
var biznoWeights = []int{1, 3, 7, 1, 3, 7, 1, 3}

// NormalizeBusinessNumber strips hyphens/spaces, verifies the 10-digit
// checksum, and returns the canonical plain form. All-identical digits
// (e.g. 0000000000) are rejected despite passing the checksum.
func NormalizeBusinessNumber(raw string) (string, error) {
	clean := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(raw), "-", ""), " ", "")
	if len(clean) != 10 {
		return "", fmt.Errorf("사업자등록번호 10자리: %w", ErrInvalidInput)
	}
	digits := make([]int, 10)
	for i := 0; i < 10; i++ {
		if clean[i] < '0' || clean[i] > '9' {
			return "", fmt.Errorf("사업자등록번호 숫자 10자리: %w", ErrInvalidInput)
		}
		digits[i] = int(clean[i] - '0')
	}
	same := true
	for i := 1; i < 10; i++ {
		if digits[i] != digits[0] {
			same = false
			break
		}
	}
	if same {
		return "", fmt.Errorf("사업자등록번호 형식 오류: %w", ErrInvalidInput)
	}
	sum := 0
	for i := 0; i < 8; i++ {
		sum += digits[i] * biznoWeights[i]
	}
	prod := digits[8] * 5
	sum += prod/10 + prod%10
	check := (10 - sum%10) % 10
	if check != digits[9] {
		return "", fmt.Errorf("사업자등록번호 체크섬 오류: %w", ErrInvalidInput)
	}
	return clean, nil
}
