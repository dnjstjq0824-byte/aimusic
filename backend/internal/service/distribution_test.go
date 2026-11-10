package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

type fakeDSPTracks struct{ m map[string]*domain.Track }

func (f fakeDSPTracks) GetByID(_ context.Context, id string) (*domain.Track, error) {
	t, ok := f.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

type fakeDSPRights struct{ m map[string]*domain.TrackRightsDeclaration }

func (f fakeDSPRights) GetByTrack(_ context.Context, trackID string) (*domain.TrackRightsDeclaration, error) {
	r, ok := f.m[trackID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

type fakeDSPPartners struct{ m map[string]*domain.PartnerProfile }

func (f fakeDSPPartners) GetByID(_ context.Context, id string) (*domain.PartnerProfile, error) {
	p, ok := f.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

type fakeDSPAlbums struct{ m map[string]*domain.Album }

func (f fakeDSPAlbums) GetByID(_ context.Context, id string) (*domain.Album, error) {
	a, ok := f.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return a, nil
}

type fakeDSPPresigner struct{}

func (fakeDSPPresigner) PresignGet(_ context.Context, bucket, key string, _ time.Duration) (string, error) {
	return "https://cdn.test/" + bucket + "/" + key, nil
}

func testDistributionService(tracks map[string]*domain.Track, rights map[string]*domain.TrackRightsDeclaration, partners map[string]*domain.PartnerProfile, albums map[string]*domain.Album) *DistributionService {
	return &DistributionService{
		tracks: fakeDSPTracks{tracks}, rights: fakeDSPRights{rights},
		partners: fakeDSPPartners{partners}, albums: fakeDSPAlbums{albums},
		storage: fakeDSPPresigner{}, stagingBucket: "staging",
	}
}

func dspReadyTrack() *domain.Track {
	isrc, bpm, key, album := "KRA122600001", 120.0, "C Major", "album-1"
	dur := 180000
	return &domain.Track{
		ID: "track-1", AlbumID: &album, PartnerID: "partner-1",
		Title: "Neon Skyline", Genre: strRef("Electronic"),
		MoodTags:         []string{"bright", "driving"},
		AIGenerationType: "FULL_AI",
		OriginalObjectKey: "staging/partner-1/abc/song.wav",
		AIProvider:        domain.ProviderSuno, AIGenerated: true,
		ISRC: &isrc, BPM: &bpm, MusicalKey: &key,
		DurationMs: &dur, Status: domain.TrackReady,
	}
}

func strRef(s string) *string { return &s }

func TestBuildDSPKitHappy(t *testing.T) {
	svc := testDistributionService(
		map[string]*domain.Track{"track-1": dspReadyTrack()},
		map[string]*domain.TrackRightsDeclaration{"track-1": {ID: "r1", TrackID: "track-1", PartnerID: "partner-1", PlanTierUsed: strRef("PRO"), SourceModel: strRef("suno-v4"), CollectiveTrustStatus: domain.CollectiveNone, Gate1PolicyPassed: true, Gate2DatePassed: true}},
		map[string]*domain.PartnerProfile{"partner-1": {ID: "partner-1", CompanyName: "Neon Works", ContactEmail: "a@b.c", Status: domain.PartnerApproved}},
		map[string]*domain.Album{"album-1": {ID: "album-1", PartnerID: "partner-1", Title: "Neon EP"}},
	)
	kit, err := svc.BuildDSPKit(context.Background(), []string{"track-1"})
	if err != nil {
		t.Fatalf("BuildDSPKit: %v", err)
	}
	if len(kit.Tracks) != 1 || len(kit.Skipped) != 0 {
		t.Fatalf("kit = %+v", kit)
	}
	got := kit.Tracks[0]
	if got.ISRC != "KRA122600001" || got.ArtistName != "Neon Works" || got.AlbumTitle != "Neon EP" {
		t.Errorf("kit track = %+v", got)
	}
	if got.BPM == nil || *got.BPM != 120.0 || got.MusicalKey != "C Major" {
		t.Errorf("analysis missing: %+v", got)
	}
	if !strings.HasPrefix(got.MasterDownloadURL, "https://cdn.test/staging/") {
		t.Errorf("master URL = %q", got.MasterDownloadURL)
	}
	if got.PlanTierUsed != "PRO" || got.SourceModel != "suno-v4" {
		t.Errorf("license summary = %+v", got)
	}
	csv := BuildDSPKitCSV(kit)
	if !strings.Contains(csv, "KRA122600001") || !strings.Contains(csv, "Neon Skyline") || !strings.HasPrefix(csv, "isrc,") {
		t.Errorf("csv:\n%s", csv)
	}
}

func TestBuildDSPKitSkipsIneligible(t *testing.T) {
	pending := dspReadyTrack()
	pending.ID = "track-2"
	pending.Status = domain.TrackUploaded
	noISRC := dspReadyTrack()
	noISRC.ID = "track-3"
	noISRC.ISRC = nil
	svc := testDistributionService(
		map[string]*domain.Track{"track-2": pending, "track-3": noISRC}, nil, nil, nil,
	)
	kit, err := svc.BuildDSPKit(context.Background(), []string{"track-2", "track-3", "missing"})
	if err != nil {
		t.Fatalf("BuildDSPKit: %v", err)
	}
	if len(kit.Tracks) != 0 || len(kit.Skipped) != 3 {
		t.Fatalf("kit = %+v", kit)
	}
}

func TestBuildDSPKitValidatesInput(t *testing.T) {
	svc := testDistributionService(nil, nil, nil, nil)
	if _, err := svc.BuildDSPKit(context.Background(), nil); err == nil {
		t.Error("empty selection must fail")
	}
	many := make([]string, 101)
	for i := range many {
		many[i] = "track-many-" + strings.Repeat("x", i+1)
	}
	if _, err := svc.BuildDSPKit(context.Background(), many); err == nil {
		t.Error("101 tracks must fail")
	}
}
