package domain

import "testing"

func TestNormalizeBusinessNumber(t *testing.T) {
	got, err := NormalizeBusinessNumber("123-45-67891")
	if err != nil {
		t.Fatalf("NormalizeBusinessNumber: %v", err)
	}
	if got != "1234567891" {
		t.Errorf("got %q", got)
	}
}

func TestNormalizeBusinessNumberRejects(t *testing.T) {
	for _, bad := range []string{"", "123", "1234567890", "abcdefghij", "123-45-6789", "0000000000"} {
		if _, err := NormalizeBusinessNumber(bad); err == nil {
			t.Errorf("input %q must fail", bad)
		}
	}
}
