package repository

import (
	"database/sql"
	"testing"
	"time"
)

func TestSplitJoinKeywordsRoundTrip(t *testing.T) {
	in := []string{"synthwave", "night drive", "ai 생성"}
	joined := joinKeywords(in)
	if got := splitKeywords(joined); len(got) != len(in) {
		t.Fatalf("round trip = %v, want %v", got, in)
	} else {
		for i := range in {
			if got[i] != in[i] {
				t.Fatalf("round trip = %v, want %v", got, in)
			}
		}
	}
}

func TestJoinKeywordsSkipsBlank(t *testing.T) {
	if got := joinKeywords([]string{"a", "  ", "", "b"}); got != "a\x1fb" {
		t.Fatalf("got %q", got)
	}
}

func TestSplitKeywordsEmpty(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		got := splitKeywords(raw)
		if len(got) != 0 {
			t.Fatalf("split(%q) = %v, want empty", raw, got)
		}
	}
}

func TestClampLimit(t *testing.T) {
	cases := []struct{ in, def, max, want int }{
		{0, 20, 100, 20},
		{-5, 20, 100, 20},
		{10, 20, 100, 10},
		{500, 20, 100, 100},
		{100, 20, 100, 100},
	}
	for _, tc := range cases {
		if got := clampLimit(tc.in, tc.def, tc.max); got != tc.want {
			t.Errorf("clampLimit(%d,%d,%d) = %d, want %d", tc.in, tc.def, tc.max, got, tc.want)
		}
	}
}

func TestNullPtrRoundTrip(t *testing.T) {
	s := "title"
	if got := nullStr(&s); got != "title" {
		t.Fatalf("nullStr = %v", got)
	}
	if got := nullStr(nil); got != nil {
		t.Fatalf("nullStr(nil) = %v", got)
	}
	n := 7
	if got := nullInt(&n); got != 7 {
		t.Fatalf("nullInt = %v", got)
	}
	if got := nullInt(nil); got != nil {
		t.Fatalf("nullInt(nil) = %v", got)
	}
	now := time.Now().Truncate(time.Second)
	if got := nullTime(&now); got != now {
		t.Fatalf("nullTime = %v", got)
	}
	if got := nullTime(nil); got != nil {
		t.Fatalf("nullTime(nil) = %v", got)
	}
	if got := strPtr(sql.NullString{String: "x", Valid: true}); got == nil || *got != "x" {
		t.Fatalf("strPtr valid = %v", got)
	}
	if got := strPtr(sql.NullString{}); got != nil {
		t.Fatalf("strPtr invalid = %v", got)
	}
	if got := intPtr(sql.NullInt64{Int64: 3, Valid: true}); got == nil || *got != 3 {
		t.Fatalf("intPtr valid = %v", got)
	}
	if got := intPtr(sql.NullInt64{}); got != nil {
		t.Fatalf("intPtr invalid = %v", got)
	}
	if got := timePtr(sql.NullTime{Time: now, Valid: true}); got == nil || !got.Equal(now) {
		t.Fatalf("timePtr valid = %v", got)
	}
	if got := timePtr(sql.NullTime{}); got != nil {
		t.Fatalf("timePtr invalid = %v", got)
	}
}
