package mail

import (
	"strings"
	"testing"
)

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"ko": "ko", "en": "en", "ja": "ja", "zh-cn": "zh", "th": "th",
		"zh-CN": "zh", "zh": "zh", "": "ko", "xx": "ko", " ko ": "ko",
	}
	for in, want := range cases {
		if got := NormalizeLocale(in); got != want {
			t.Fatalf("NormalizeLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderVerifyAllLocales(t *testing.T) {
	link := "https://app.example.test/verify-email?token=tok-1"
	seen := map[string]bool{}
	for _, loc := range []string{"ko", "en", "ja", "zh-cn", "th"} {
		subject, body := RenderVerify(loc, link, 24)
		if subject == "" || body == "" {
			t.Fatalf("locale %s rendered empty", loc)
		}
		if !strings.Contains(subject, "AVIBE") {
			t.Fatalf("locale %s subject = %q", loc, subject)
		}
		if !strings.Contains(body, link) || !strings.Contains(body, "24") {
			t.Fatalf("locale %s body missing link/ttl: %q", loc, body)
		}
		if strings.Contains(body, "%!") {
			t.Fatalf("locale %s body has bad verb: %q", loc, body)
		}
		seen[subject] = true
	}
	if len(seen) != 5 {
		t.Fatalf("subjects not distinct per locale: %v", seen)
	}
}

func TestRenderResetAllLocales(t *testing.T) {
	link := "https://app.example.test/reset-password?token=tok-1"
	seen := map[string]bool{}
	for _, loc := range []string{"ko", "en", "ja", "zh-cn", "th"} {
		subject, body := RenderReset(loc, link, 30)
		if subject == "" || body == "" {
			t.Fatalf("locale %s rendered empty", loc)
		}
		if !strings.Contains(body, link) || !strings.Contains(body, "30") {
			t.Fatalf("locale %s body missing link/ttl: %q", loc, body)
		}
		if strings.Contains(body, "%!") {
			t.Fatalf("locale %s body has bad verb: %q", loc, body)
		}
		seen[subject] = true
	}
	if len(seen) != 5 {
		t.Fatalf("subjects not distinct per locale: %v", seen)
	}
}

func TestRenderFallsBackToKo(t *testing.T) {
	for _, render := range []func(string, string, int) (string, string){
		RenderVerify, RenderReset,
	} {
		koSubject, _ := render("ko", "link", 1)
		badSubject, badBody := render("xx", "link", 1)
		if badSubject != koSubject || !strings.Contains(badBody, "link") {
			t.Fatalf("fallback = %q / %q", badSubject, badBody)
		}
	}
}
