package search

import "testing"

func TestRegistryParity(t *testing.T) {
	if len(Registry) != 19 {
		t.Fatalf("want 19 canonical keys, got %d", len(Registry))
	}
	seen := map[string]bool{}
	for _, tag := range Registry {
		if seen[tag.Key] {
			t.Fatalf("duplicate key %q", tag.Key)
		}
		seen[tag.Key] = true
		if len(tag.Facets) == 0 {
			t.Fatalf("key %q has no facet", tag.Key)
		}
		for _, loc := range []string{"ko", "en", "ja", "zh-cn", "th"} {
			if tag.Names[loc] == "" {
				t.Fatalf("key %q missing %s display name", tag.Key, loc)
			}
		}
	}
}

func TestKeysForQuery(t *testing.T) {
	cases := []struct {
		q    string
		want []string
	}{
		{"focus", []string{"focus"}},
		{"Focus", []string{"focus"}},
		{"집중", []string{"focus"}},
		{"集中", []string{"focus"}},
		{"专注", []string{"focus"}},
		{"โฟกัส", []string{"focus"}},
		{"카페", []string{"cafe"}},
		{"电子", []string{"electronic"}},
		{"Electronic", []string{"electronic"}},
		{"ELECTRONIC", []string{"electronic"}},
		{"Gym", []string{"gym"}},
		{"헬스장", []string{"gym"}},
		{"midnight neon", nil},
		{"", nil},
		{"   ", nil},
	}
	for _, tc := range cases {
		got := KeysForQuery(tc.q)
		if len(got) != len(tc.want) {
			t.Fatalf("q %q: got %v, want %v", tc.q, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("q %q: got %v, want %v", tc.q, got, tc.want)
			}
		}
	}
}

func TestIsKeyAndFacets(t *testing.T) {
	if !IsKey("Chill") || !IsKey("ambient") {
		t.Fatal("known keys must validate case-insensitively")
	}
	if IsKey("polka") {
		t.Fatal("unknown key must not validate")
	}
	f := FacetsFor("cafe")
	if len(f) != 2 {
		t.Fatalf("cafe must carry mood+usecase, got %v", f)
	}
	if FacetsFor("polka") != nil {
		t.Fatal("unknown key must have no facets")
	}
}
