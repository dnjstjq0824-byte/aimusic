package domain

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, fallback, want string
	}{
		{"Midnight Drive", "track", "midnight-drive"},
		{"  Café   BGM  ", "track", "café-bgm"},
		{"밤 산책 / Chill", "track", "밤-산책-chill"},
		{"a?b#c%d&e+f=g", "track", "a-b-c-d-e-f-g"},
		{"", "playlist", "playlist"},
		{"   ", "track", "track"},
		{"---", "track", "track"},
		{"Lofi Beats हो", "track", "lofi-beats-हो"},
		{"เพลงช้าๆ ฟังสบาย", "track", "เพลงช้าๆ-ฟังสบาย"},
	}
	for _, c := range cases {
		if got := Slugify(c.in, c.fallback); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	long := strings.Repeat("a", 200)
	if got := Slugify(long, "track"); len([]rune(got)) != 80 {
		t.Errorf("long slug runes = %d, want 80", len([]rune(got)))
	}
}

func TestNewShortIDFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		id, err := NewShortID()
		if err != nil {
			t.Fatalf("NewShortID: %v", err)
		}
		if !ValidateShortID(id) {
			t.Fatalf("invalid short id %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate short id %q in 500 draws", id)
		}
		seen[id] = true
	}
	for _, bad := range []string{"", "abc", "abcdefghi", "abcd-efg", "!!!!!!!!"} {
		if ValidateShortID(bad) {
			t.Errorf("ValidateShortID(%q) = true, want false", bad)
		}
	}
}

func TestValidateHandle(t *testing.T) {
	for _, good := range []string{"abc", "creator_01", "a1_", "x" + strings.Repeat("y", 28) + "z"} {
		if err := ValidateHandle(good); err != nil {
			t.Errorf("ValidateHandle(%q) = %v, want nil", good, err)
		}
	}
	for _, bad := range []string{"", "ab", strings.Repeat("a", 31), "Abc", "a-b", "a b", "a.b", "tracks", "admin", "api", "avibe", "null"} {
		if err := ValidateHandle(bad); err == nil {
			t.Errorf("ValidateHandle(%q) = nil, want error", bad)
		}
	}
}

func TestSplitSlugID(t *testing.T) {
	uuid := "123e4567-e89b-12d3-a456-426614174000"
	if _, _, kind := SplitSlugID(uuid); kind != SlugKindUUID {
		t.Errorf("uuid kind = %q", kind)
	}
	if _, id, kind := SplitSlugID("Ab3xK9pQ"); kind != SlugKindShortID || id != "Ab3xK9pQ" {
		t.Errorf("short kind = %q id = %q", kind, id)
	}
	slug, id, kind := SplitSlugID("midnight-drive-Ab3xK9pQ")
	if kind != SlugKindSlugShort || slug != "midnight-drive" || id != "Ab3xK9pQ" {
		t.Errorf("slugshort = %q %q %q", slug, id, kind)
	}
	slug, id, kind = SplitSlugID("밤-산책-Ab3xK9pQ")
	if kind != SlugKindSlugShort || slug != "밤-산책" || id != "Ab3xK9pQ" {
		t.Errorf("unicode slugshort = %q %q %q", slug, id, kind)
	}
	for _, bad := range []string{"", "nope", "slug-toolongtail", "Ab3xK9pQ-extra-semis"} {
		if _, _, kind := SplitSlugID(bad); kind != SlugKindInvalid {
			t.Errorf("SplitSlugID(%q) kind = %q, want invalid", bad, kind)
		}
	}
}

func TestAnalyticsAllowlist(t *testing.T) {
	for _, name := range []string{"signup_started", "play_30s", "inquiry_submitted", "share_clicked", "embed_loaded", "web_vital"} {
		if !ValidAnalyticsEvent(name) {
			t.Errorf("ValidAnalyticsEvent(%q) = false", name)
		}
	}
	if ValidAnalyticsEvent("keystroke") {
		t.Error("ValidAnalyticsEvent(keystroke) = true, want false")
	}
	props := SanitizeEventProps(map[string]string{
		"channel": "kakao", "email": "a@b.c", "token": "secret", "locale": "ko",
	})
	if _, ok := props["channel"]; !ok {
		t.Error("channel dropped, want kept")
	}
	if _, ok := props["email"]; ok {
		t.Error("email kept, want dropped")
	}
	if _, ok := props["token"]; ok {
		t.Error("token kept, want dropped")
	}
}

func TestIsSEOLocale(t *testing.T) {
	for _, l := range []string{"ko", "en", "ja", "zh-cn", "th", "EN", " zh-cn "} {
		if !IsSEOLocale(l) {
			t.Errorf("IsSEOLocale(%q) = false", l)
		}
	}
	for _, l := range []string{"", "zh-tw", "fr", "zh_CN"} {
		if IsSEOLocale(l) {
			t.Errorf("IsSEOLocale(%q) = true", l)
		}
	}
}
