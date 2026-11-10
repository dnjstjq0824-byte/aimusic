package service

import (
	"strings"
	"testing"
)

// P2-6 (#24): per-track certificate numbers are unguessable public IDs:
// AVB- + 10 unambiguous upper alphanumerics (no 0/O/1/I).

func TestGenerateCertNumberFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		n, err := GenerateCertNumber()
		if err != nil {
			t.Fatalf("GenerateCertNumber: %v", err)
		}
		if !ValidCertNumber(n) {
			t.Fatalf("generated %q fails validation", n)
		}
		if seen[n] {
			t.Fatalf("duplicate number %q in 50 draws", n)
		}
		seen[n] = true
		if strings.ContainsAny(n, "0O1I") {
			t.Fatalf("ambiguous char in %q", n)
		}
	}
}

func TestValidCertNumberRejects(t *testing.T) {
	for _, bad := range []string{"", "AVB-", "AVB-ABC", "AVB-ABCDEFGHIJK", "avb-ABCDEFGHJK", "AVB-ABCD EFGH", "AVB-ABCD-EFGH", "XYZ-ABCDEFGHJK"} {
		if ValidCertNumber(bad) {
			t.Fatalf("%q must be invalid", bad)
		}
	}
	if !ValidCertNumber("AVB-ABCDEFGHJK") {
		t.Fatalf("well-formed number must validate")
	}
}
