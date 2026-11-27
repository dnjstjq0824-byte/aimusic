package service

import (
	"errors"
	"strings"
	"testing"

	"avibe-platform/backend/internal/domain"
)

func TestNormalizePlaylistFields(t *testing.T) {
	desc := "  late-night drive  "
	title, d, err := normalizePlaylistFields("  Night Mix  ", &desc)
	if err != nil || title != "Night Mix" || d == nil || *d != "late-night drive" {
		t.Fatalf("trim: got %q %v %v", title, d, err)
	}
	if _, d, _ := normalizePlaylistFields("x", nil); d != nil {
		t.Fatal("nil description must stay nil (update: unchanged)")
	}
	if _, _, err := normalizePlaylistFields("   ", nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("blank title: %v", err)
	}
	// Limits are runes, not bytes: 100 Hangul syllables are allowed.
	if _, _, err := normalizePlaylistFields(strings.Repeat("가", domain.PlaylistTitleMax), nil); err != nil {
		t.Fatalf("100-rune title rejected: %v", err)
	}
	if _, _, err := normalizePlaylistFields(strings.Repeat("가", domain.PlaylistTitleMax+1), nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("101-rune title accepted")
	}
	long := strings.Repeat("a", domain.PlaylistDescriptionMax+1)
	if _, _, err := normalizePlaylistFields("t", &long); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("over-long description accepted")
	}
}

func TestValidateReorderIDsPermutation(t *testing.T) {
	cur := []string{"a", "b", "c"}
	if err := validateReorderIDs(cur, []string{"c", "a", "b"}); err != nil {
		t.Fatalf("valid permutation rejected: %v", err)
	}
	for _, bad := range [][]string{{"a", "b"}, {"a", "a", "b"}, {"a", "b", "z"}} {
		if err := validateReorderIDs(cur, bad); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%v accepted", bad)
		}
	}
}

func TestNormalizeCommentBody(t *testing.T) {
	if b, err := normalizeCommentBody("  hi  "); err != nil || b != "hi" {
		t.Fatalf("trim: %q %v", b, err)
	}
	if _, err := normalizeCommentBody(" \n\t "); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("blank comment accepted")
	}
	if _, err := normalizeCommentBody(strings.Repeat("가", domain.TrackCommentMax)); err != nil {
		t.Fatalf("max-length comment rejected: %v", err)
	}
	if _, err := normalizeCommentBody(strings.Repeat("가", domain.TrackCommentMax+1)); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("over-long comment accepted")
	}
}

func TestNormalizeListingPatch(t *testing.T) {
	moods := []string{" chill ", "Chill", "", "focus"}
	p, err := normalizeListingPatch(TrackListingPatch{MoodTags: &moods})
	if err != nil || len(*p.MoodTags) != 2 || (*p.MoodTags)[0] != "chill" || (*p.MoodTags)[1] != "focus" {
		t.Fatalf("dedupe/trim moods: %v %v", p.MoodTags, err)
	}
	many := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	if _, err := normalizeListingPatch(TrackListingPatch{MoodTags: &many}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("9 moods accepted")
	}
	g := strings.Repeat("x", studioGenreMax+1)
	if _, err := normalizeListingPatch(TrackListingPatch{Genre: &g}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("over-long genre accepted")
	}
	if p, err := normalizeListingPatch(TrackListingPatch{}); err != nil || p.Genre != nil || p.MoodTags != nil {
		t.Fatal("empty patch must stay empty")
	}
}

func TestEffectiveCommentLimit(t *testing.T) {
	for in, want := range map[int]int{0: 20, -1: 20, 10: 10, 50: 50, 500: 50} {
		if got := effectiveCommentLimit(in); got != want {
			t.Fatalf("limit %d: got %d want %d", in, got, want)
		}
	}
}
