package domain

import "testing"

func TestNormalizeISRC(t *testing.T) {
	got, err := NormalizeISRC("kr-a12-26-00001")
	if err != nil {
		t.Fatalf("NormalizeISRC: %v", err)
	}
	if got != "KRA122600001" {
		t.Errorf("got %q, want KRA122600001", got)
	}
	got, err = NormalizeISRC("KRA122600001")
	if err != nil || got != "KRA122600001" {
		t.Errorf("plain code: got %q err %v", got, err)
	}
}

func TestNormalizeISRCRejects(t *testing.T) {
	for _, bad := range []string{"", "KR-A1-26-1", "KRA122600001X", "12A122600001", "KR-!!!-26-00001"} {
		if _, err := NormalizeISRC(bad); err == nil {
			t.Errorf("input %q must fail", bad)
		}
	}
}

func TestValidISRC(t *testing.T) {
	if !ValidISRC("KRA122600001") {
		t.Error("canonical code must validate")
	}
	if ValidISRC("short") {
		t.Error("short code must not validate")
	}
}
