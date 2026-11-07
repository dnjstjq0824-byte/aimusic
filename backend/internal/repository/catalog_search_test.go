package repository

import (
	"strings"
	"testing"
)

func TestTrackSortOrderByWhitelist(t *testing.T) {
	want := map[string]string{
		"":        "created_at DESC",
		"recent":  "created_at DESC",
		"oldest":  "created_at ASC",
		"popular": "play_count DESC, created_at DESC",
		"title":   "title ASC, created_at DESC",
	}
	for sort, orderBy := range want {
		got, ok := TrackSortOrderBy(sort)
		if !ok || got != orderBy {
			t.Fatalf("sort %q: got (%q, %v), want (%q, true)", sort, got, ok, orderBy)
		}
	}
	badSorts := []string{"created_at DESC; DROP TABLE tracks", "plays", "RANDOM()", "recent "}
	for _, bad := range badSorts {
		if _, ok := TrackSortOrderBy(bad); ok {
			t.Fatalf("sort %q accepted, must be rejected", bad)
		}
	}
}

func TestBuildTrackSearchEmpty(t *testing.T) {
	where, args := buildTrackSearch(TrackFilter{})
	if where != " WHERE status = 'READY'" {
		t.Fatalf("unexpected where: %q", where)
	}
	if len(args) != 0 {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestBuildTrackSearchAllFilters(t *testing.T) {
	where, args := buildTrackSearch(TrackFilter{
		Q: "midnight", Provider: "suno", Genre: "Lo-Fi", Mood: "Chill",
	})
	if len(args) != 4 {
		t.Fatalf("want 4 args, got %v", args)
	}
	if args[0] != "%midnight%" || args[1] != "SUNO" || args[2] != "lo-fi" || args[3] != "chill" {
		t.Fatalf("arg mismatch: %v", args)
	}
	for _, frag := range []string{
		"status = 'READY'",
		"title ILIKE $1 ESCAPE '\\'",
		"ai_provider = $2",
		"LOWER(genre) = $3",
		"unnest(mood_tags)",
		"LOWER(mt) = $4",
	} {
		if !strings.Contains(where, frag) {
			t.Fatalf("where %q missing %q", where, frag)
		}
	}
}

func TestBuildTrackSearchEscapesLikeWildcards(t *testing.T) {
	where, args := buildTrackSearch(TrackFilter{Q: `100%_real\hit`})
	if len(args) != 1 {
		t.Fatalf("want 1 arg, got %v", args)
	}
	// Wildcards must match literally, so they arrive escaped inside the pattern.
	if args[0] != `%100\%\_real\\hit%` {
		t.Fatalf("wildcard escaping mismatch: %q", args[0])
	}
	if !strings.Contains(where, `ESCAPE '\'`) {
		t.Fatalf("where %q missing ESCAPE clause", where)
	}
}

func TestBuildTrackSearchTrimsAndSkipsBlanks(t *testing.T) {
	where, args := buildTrackSearch(TrackFilter{Q: "  ", Provider: " ", Genre: "", Mood: "	"})
	if where != " WHERE status = 'READY'" || len(args) != 0 {
		t.Fatalf("blank filters must be skipped: %q %v", where, args)
	}
}
