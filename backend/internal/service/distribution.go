package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/licensegate"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
)

// MaxDSPKitTracks caps one export selection (URL + signing cost bound).
const MaxDSPKitTracks = 100

// DSPMasterURLTTL is the presigned master-download lifetime.
const DSPMasterURLTTL = time.Hour

// dspTrackStore is the distribution track surface.
type dspTrackStore interface {
	GetByID(ctx context.Context, id string) (*domain.Track, error)
}

// dspRightsStore is the distribution rights surface.
type dspRightsStore interface {
	GetByTrack(ctx context.Context, trackID string) (*domain.TrackRightsDeclaration, error)
}

// dspPartnerStore is the distribution partner surface.
type dspPartnerStore interface {
	GetByID(ctx context.Context, id string) (*domain.PartnerProfile, error)
}

// dspAlbumStore is the distribution album surface.
type dspAlbumStore interface {
	GetByID(ctx context.Context, id string) (*domain.Album, error)
}

// dspPresigner signs master downloads.
type dspPresigner interface {
	PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
}

// DSPKitTrack is one distributor-ready track row.
type DSPKitTrack struct {
	TrackID           string   `json:"trackId"`
	Title             string   `json:"title"`
	ArtistName        string   `json:"artistName"`
	AlbumTitle        string   `json:"albumTitle,omitempty"`
	Genre             string   `json:"genre,omitempty"`
	MoodTags          []string `json:"moodTags,omitempty"`
	DurationMs        int      `json:"durationMs,omitempty"`
	ISRC              string   `json:"isrc"`
	BPM               *float64 `json:"bpm,omitempty"`
	MusicalKey        string   `json:"musicalKey,omitempty"`
	AIProvider        string   `json:"aiProvider"`
	AIGenerated       bool     `json:"aiGenerated"`
	AIDisclosureText  string   `json:"aiDisclosureText,omitempty"`
	PlanTierUsed      string   `json:"planTierUsed,omitempty"`
	SourceModel       string   `json:"sourceModel,omitempty"`
	CertPolicyVersion string   `json:"certPolicyVersion,omitempty"`
	MasterObjectKey   string   `json:"masterObjectKey"`
	MasterDownloadURL string   `json:"masterDownloadUrl"`
	MasterURLExpiry   string   `json:"masterUrlExpiry"`
}

// DSPKitSkip explains an excluded selection.
type DSPKitSkip struct {
	TrackID string `json:"trackId"`
	Reason  string `json:"reason"`
}

// DSPKit is the distributor handoff bundle (WI-05 #32, local-first).
type DSPKit struct {
	GeneratedAt string        `json:"generatedAt"`
	Tracks      []DSPKitTrack `json:"tracks"`
	Skipped     []DSPKitSkip  `json:"skipped"`
}

// DistributionService builds distributor handoff kits for explicitly
// selected READY tracks with assigned ISRCs.
type DistributionService struct {
	tracks        dspTrackStore
	rights        dspRightsStore
	partners      dspPartnerStore
	albums        dspAlbumStore
	storage       dspPresigner
	stagingBucket string
	now           func() time.Time
}

// NewDistributionService wires the service over live repositories.
func NewDistributionService(repos *repository.Repos, st storage.Storage, stagingBucket string) *DistributionService {
	return &DistributionService{
		tracks: repos.Tracks, rights: repos.Rights,
		partners: repos.Partners, albums: repos.Albums,
		storage: st, stagingBucket: stagingBucket, now: time.Now,
	}
}

// BuildDSPKit assembles the kit; ineligible selections are skipped with
// reasons instead of failing the whole export.
func (s *DistributionService) BuildDSPKit(ctx context.Context, trackIDs []string) (*DSPKit, error) {
	seen := map[string]bool{}
	ids := make([]string, 0, len(trackIDs))
	for _, id := range trackIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: trackIds 1개 이상", domain.ErrInvalidInput)
	}
	if len(ids) > MaxDSPKitTracks {
		return nil, fmt.Errorf("%w: 최대 %d곡", domain.ErrInvalidInput, MaxDSPKitTracks)
	}
	nowFn := s.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	kit := &DSPKit{
		GeneratedAt: now.Format(time.RFC3339),
		Tracks:      []DSPKitTrack{},
		Skipped:     []DSPKitSkip{},
	}
	for _, id := range ids {
		row, skip := s.buildRow(ctx, id, now)
		if skip != "" {
			kit.Skipped = append(kit.Skipped, DSPKitSkip{TrackID: id, Reason: skip})
			continue
		}
		kit.Tracks = append(kit.Tracks, *row)
	}
	return kit, nil
}

func (s *DistributionService) buildRow(ctx context.Context, id string, now time.Time) (*DSPKitTrack, string) {
	track, err := s.tracks.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "track_not_found"
		}
		return nil, "track_lookup_error"
	}
	if track.Status != domain.TrackReady {
		return nil, "not_ready:" + track.Status
	}
	if track.ISRC == nil || *track.ISRC == "" {
		return nil, "missing_isrc"
	}
	rights, err := s.rights.GetByTrack(ctx, id)
	if err != nil {
		return nil, "rights_missing"
	}
	partner, err := s.partners.GetByID(ctx, track.PartnerID)
	if err != nil {
		return nil, "partner_missing"
	}
	row := &DSPKitTrack{
		TrackID: track.ID, Title: track.Title, ArtistName: partner.CompanyName,
		MoodTags: track.MoodTags, ISRC: *track.ISRC, BPM: track.BPM,
		AIProvider: track.AIProvider, AIGenerated: track.AIGenerated,
		MasterObjectKey: track.OriginalObjectKey,
	}
	if track.Genre != nil {
		row.Genre = *track.Genre
	}
	if track.DurationMs != nil {
		row.DurationMs = *track.DurationMs
	}
	if track.MusicalKey != nil {
		row.MusicalKey = *track.MusicalKey
	}
	if track.AIDisclosureText != nil {
		row.AIDisclosureText = *track.AIDisclosureText
	}
	if track.AlbumID != nil {
		if album, err := s.albums.GetByID(ctx, *track.AlbumID); err == nil {
			row.AlbumTitle = album.Title
		}
	}
	if rights.PlanTierUsed != nil {
		row.PlanTierUsed = *rights.PlanTierUsed
	}
	if rights.SourceModel != nil {
		row.SourceModel = *rights.SourceModel
	}
	if rights.CertSnapshotJSON != nil {
		if snap, err := licensegate.ParseCertSnapshot(*rights.CertSnapshotJSON); err == nil {
			row.CertPolicyVersion = snap.PolicyVersion
		}
	}
	signed, err := s.storage.PresignGet(ctx, s.stagingBucket, track.OriginalObjectKey, DSPMasterURLTTL)
	if err != nil {
		return nil, "master_sign_error"
	}
	row.MasterDownloadURL = signed
	row.MasterURLExpiry = now.Add(DSPMasterURLTTL).Format(time.RFC3339)
	return row, ""
}

// BuildDSPKitCSV flattens the kit for distributor spreadsheets.
func BuildDSPKitCSV(kit *DSPKit) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"isrc", "title", "artist", "album", "genre", "moods", "durationMs",
		"bpm", "musicalKey", "aiProvider", "aiGenerated", "aiDisclosure", "planTier", "sourceModel",
		"certPolicyVersion", "masterObjectKey", "masterDownloadUrl", "masterUrlExpiry"})
	for _, t := range kit.Tracks {
		bpm := ""
		if t.BPM != nil {
			bpm = strconv.FormatFloat(*t.BPM, 'f', 1, 64)
		}
		_ = w.Write([]string{t.ISRC, t.Title, t.ArtistName, t.AlbumTitle, t.Genre,
			strings.Join(t.MoodTags, ";"), strconv.Itoa(t.DurationMs), bpm, t.MusicalKey,
			t.AIProvider, strconv.FormatBool(t.AIGenerated), t.AIDisclosureText,
			t.PlanTierUsed, t.SourceModel, t.CertPolicyVersion,
			t.MasterObjectKey, t.MasterDownloadURL, t.MasterURLExpiry})
	}
	w.Flush()
	return buf.String()
}
