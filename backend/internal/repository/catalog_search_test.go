package repository

import (
	"strings"
	"testing"
)

func TestTrackSortOrderByWhitelist(t *testing.T) {
	want := map[string]string{
		"":        "tracks.created_at DESC",
		"recent":  "tracks.created_at DESC",
		"oldest":  "tracks.created_at ASC",
		"popular": "tracks.play_count DESC, tracks.created_at DESC",
		"title":   "tracks.title ASC, tracks.created_at DESC",
	}
	for sort, orderBy := range want {
		got, ok := TrackSortOrderBy(sort)
		if !ok || got != orderBy {
			t.Fatalf("sort %q: got (%q, %v), want (%q, true)", sort, got, ok, orderBy)
		}
	}
	badSorts := []string{"created_at DESC; DROP TABLE tracks", "plays", "RANDOM()", "recent ", "relevant"}
	for _, bad := range badSorts {
		if _, ok := TrackSortOrderBy(bad); ok {
			t.Fatalf("sort %q accepted, must be rejected", bad)
		}
	}
}

func TestBuildTrackSearchEmpty(t *testing.T) {
	where, args, simArg := buildTrackSearch(TrackFilter{})
	if where != " WHERE tracks.status = 'READY'" {
		t.Fatalf("unexpected where: %q", where)
	}
	if len(args) != 0 {
		t.Fatalf("unexpected args: %v", args)
	}
	if simArg != 0 {
		t.Fatalf("empty q must yield simArg 0, got %d", simArg)
	}
}

func TestBuildTrackSearchAllFilters(t *testing.T) {
	where, args, simArg := buildTrackSearch(TrackFilter{
		Q: "midnight", Provider: "suno", Genre: "Lo-Fi", Mood: "Chill",
		Certified: true, PartnerID: "b0000000-0000-4000-8000-000000000001",
		MinBpm: 90, MaxBpm: 120, MinDurMs: 60000, MaxDurMs: 300000,
	})
	// pattern, raw, provider, genre, mood, partner, minBpm, maxBpm, minDur, maxDur
	if len(args) != 10 {
		t.Fatalf("want 10 args, got %v", args)
	}
	if args[0] != "%midnight%" || args[1] != "midnight" || args[2] != "SUNO" || args[3] != "lo-fi" || args[4] != "chill" {
		t.Fatalf("arg mismatch: %v", args)
	}
	if simArg != 2 {
		t.Fatalf("simArg must point at the raw q arg, got %d", simArg)
	}
	for _, frag := range []string{
		"tracks.status = 'READY'",
		"tracks.title ILIKE $1 ESCAPE '\\'",
		"u.display_name ILIKE $1 ESCAPE '\\'",
		"tracks.title % $2",
		"u.display_name % $2",
		"tracks.ai_provider = $3",
		"LOWER(tracks.genre) = $4",
		"unnest(tracks.mood_tags)",
		"LOWER(mt) = $5",
		"track_certificates tc",
		"tracks.partner_id = $6::uuid",
		"tracks.bpm >= $7",
		"tracks.bpm <= $8",
		"tracks.duration_ms >= $9",
		"tracks.duration_ms <= $10",
	} {
		if !strings.Contains(where, frag) {
			t.Fatalf("where %q missing %q", where, frag)
		}
	}
}

func TestBuildTrackSearchTagExpansion(t *testing.T) {
	where, args, _ := buildTrackSearch(TrackFilter{Q: "집중"})
	found := false
	for _, a := range args {
		if a == "focus" {
			found = true
		}
	}
	if !found {
		t.Fatalf("display-name query must expand to the key: %v", args)
	}
	if !strings.Contains(where, "LOWER(mt) IN (") || !strings.Contains(where, "LOWER(tracks.genre) IN (") {
		t.Fatalf("tag expansion missing from %q", where)
	}
}

func TestRelevanceOrderBy(t *testing.T) {
	got := relevanceOrderBy(2)
	want := "GREATEST(similarity(tracks.title,$2),similarity(u.display_name,$2)) DESC, tracks.created_at DESC"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildTrackSearchEscapesLikeWildcards(t *testing.T) {
	where, args, _ := buildTrackSearch(TrackFilter{Q: `100%_real\hit`})
	if len(args) != 2 {
		t.Fatalf("want 2 args, got %v", args)
	}
	// Wildcards must match literally, so they arrive escaped inside the pattern.
	if args[0] != `%100\%\_real\\hit%` {
		t.Fatalf("wildcard escaping mismatch: %q", args[0])
	}
	if args[1] != `100%_real\hit` {
		t.Fatalf("raw q arg mismatch: %q", args[1])
	}
	if !strings.Contains(where, `ESCAPE '\'`) {
		t.Fatalf("where %q missing ESCAPE clause", where)
	}
}

func TestBuildTrackSearchTrimsAndSkipsBlanks(t *testing.T) {
	where, args, _ := buildTrackSearch(TrackFilter{Q: "  ", Provider: " ", Genre: "", Mood: "	"})
	if where != " WHERE tracks.status = 'READY'" || len(args) != 0 {
		t.Fatalf("blank filters must be skipped: %q %v", where, args)
	}
}
