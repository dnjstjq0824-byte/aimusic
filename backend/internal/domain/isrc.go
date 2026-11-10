package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// isrcCanonical matches a hyphen-stripped ISRC: 2-letter country, 3-char
// registrant, 2-digit year, 5-digit designation (e.g. KRA122600001).
var isrcCanonical = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{3}[0-9]{7}$`)

// NormalizeISRC strips hyphens/spaces, uppercases, and validates an ISRC,
// returning the 12-char canonical form for storage.
func NormalizeISRC(raw string) (string, error) {
	clean := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(raw), "-", ""), " ", ""))
	if !isrcCanonical.MatchString(clean) {
		return "", fmt.Errorf("ISRC 형식 오류 (예: KR-A12-26-00001): %w", ErrInvalidInput)
	}
	return clean, nil
}

// ValidISRC reports whether s is a canonical 12-char ISRC.
func ValidISRC(s string) bool {
	return isrcCanonical.MatchString(s)
}
