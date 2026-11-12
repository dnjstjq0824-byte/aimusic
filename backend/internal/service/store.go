package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// Store pairing/counting rules (WI-05 #19/#20/#21).
const (
	StorePairingCodeLen  = 8
	StorePairingTTL      = 15 * time.Minute
	StoreTokenBytes      = 32
	StoreCountSeconds    = 30
	MaxStoreLicenses     = 100
	MaxStoreNameChars    = 60
	MaxDeviceNameChars   = 40
	storePairingAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

// storeStore is the store persistence surface.
type storeStore interface {
	CreateBusiness(ctx context.Context, partnerID, businessNumber, storeName string, licenseCount int) (*domain.StoreBusiness, error)
	GetBusinessByID(ctx context.Context, id string) (*domain.StoreBusiness, error)
	ListBusinessesByPartner(ctx context.Context, partnerID string) ([]*domain.StoreBusiness, error)
	VerifyBusiness(ctx context.Context, id, status, verifierID string) error
	CreateDevice(ctx context.Context, businessID, deviceName, pairingCode string, pairingExpiry time.Time) (*domain.StoreDevice, error)
	GetDeviceByID(ctx context.Context, id string) (*domain.StoreDevice, error)
	GetDeviceByPairingCode(ctx context.Context, code string) (*domain.StoreDevice, error)
	GetDeviceByTokenHash(ctx context.Context, hash string) (*domain.StoreDevice, error)
	ListDevicesByBusiness(ctx context.Context, businessID string) ([]*domain.StoreDevice, error)
	ActivateDevice(ctx context.Context, id, tokenHash string) error
	UpdateDevice(ctx context.Context, id, deviceName string, volume int) error
	RevokeDevice(ctx context.Context, id string) error
	TouchDeviceSeen(ctx context.Context, id string) error
	UpsertSchedule(ctx context.Context, businessID string, weekday int, open, closeTime string, playlistID *string, isClosed bool) (*domain.StoreSchedule, error)
	ListSchedulesByBusiness(ctx context.Context, businessID string) ([]*domain.StoreSchedule, error)
	AppendPlayLog(ctx context.Context, deviceID, trackID string, durationSecs int, counted bool) (*domain.StorePlayLog, error)
	ListPlayLogsByDevice(ctx context.Context, deviceID string, limit int) ([]*domain.StorePlayLog, error)
}

type storeTrackStore interface {
	GetByID(ctx context.Context, id string) (*domain.Track, error)
}

type storePartnerStore interface {
	GetByID(ctx context.Context, id string) (*domain.PartnerProfile, error)
}

type storePlaylistStore interface {
	GetByID(ctx context.Context, id string) (*domain.Playlist, error)
	ListTracks(ctx context.Context, playlistID string) ([]*domain.PlaylistTrack, error)
	ListOfficialCuration(ctx context.Context) ([]*domain.Playlist, error)
}

// StoreService runs store businesses, player devices, schedules, and
// certified play counting.
type StoreService struct {
	store     storeStore
	tracks    storeTrackStore
	partners  storePartnerStore
	playlists storePlaylistStore
	now       func() time.Time
}

// NewStoreService wires the service over live repositories.
func NewStoreService(repos *repository.Repos) *StoreService {
	return &StoreService{
		store: repos.Store, tracks: repos.Tracks,
		partners: repos.Partners, playlists: repos.Playlists, now: time.Now,
	}
}

func (s *StoreService) clock() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

// RegisterBusiness files a store business for admin verification.
func (s *StoreService) RegisterBusiness(ctx context.Context, partnerID, businessNumber, storeName string, licenseCount int) (*domain.StoreBusiness, error) {
	bizNo, err := domain.NormalizeBusinessNumber(businessNumber)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(storeName)
	if name == "" || len([]rune(name)) > MaxStoreNameChars {
		return nil, fmt.Errorf("%w: 매장명 1..60자", domain.ErrInvalidInput)
	}
	if licenseCount < 1 || licenseCount > MaxStoreLicenses {
		return nil, fmt.Errorf("%w: 라이선스 1..100", domain.ErrInvalidInput)
	}
	return s.store.CreateBusiness(ctx, partnerID, bizNo, name, licenseCount)
}

// ListBusinesses returns the partner's store businesses.
func (s *StoreService) ListBusinesses(ctx context.Context, partnerID string) ([]*domain.StoreBusiness, error) {
	return s.store.ListBusinessesByPartner(ctx, partnerID)
}

// VerifyBusiness approves or suspends a store business (admin).
func (s *StoreService) VerifyBusiness(ctx context.Context, businessID, verifierID string, approve bool) (*domain.StoreBusiness, error) {
	status := domain.StoreSuspended
	if approve {
		status = domain.StoreApproved
	}
	if err := s.store.VerifyBusiness(ctx, businessID, status, verifierID); err != nil {
		return nil, err
	}
	return s.store.GetBusinessByID(ctx, businessID)
}

func (s *StoreService) ownBusiness(ctx context.Context, partnerID, businessID string) (*domain.StoreBusiness, error) {
	b, err := s.store.GetBusinessByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	if b.PartnerID != partnerID {
		return nil, domain.ErrForbidden
	}
	return b, nil
}

// CreateDevice registers a player device and returns it with a pairing code.
func (s *StoreService) CreateDevice(ctx context.Context, partnerID, businessID, deviceName string) (*domain.StoreDevice, error) {
	b, err := s.ownBusiness(ctx, partnerID, businessID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(deviceName)
	if name == "" || len([]rune(name)) > MaxDeviceNameChars {
		return nil, fmt.Errorf("%w: 기기명 1..40자", domain.ErrInvalidInput)
	}
	devices, err := s.store.ListDevicesByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}
	live := 0
	for _, d := range devices {
		if d.Status != domain.DeviceRevoked {
			live++
		}
	}
	if live >= b.LicenseCount {
		return nil, fmt.Errorf("%w: 라이선스 초과(매장당 %d대)", domain.ErrForbidden, b.LicenseCount)
	}
	code, err := randomPairingCode()
	if err != nil {
		return nil, err
	}
	return s.store.CreateDevice(ctx, businessID, name, code, s.clock().Add(StorePairingTTL))
}

func randomPairingCode() (string, error) {
	raw := make([]byte, StorePairingCodeLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, StorePairingCodeLen)
	for i, b := range raw {
		out[i] = storePairingAlphabet[int(b)%len(storePairingAlphabet)]
	}
	return string(out), nil
}

// HashDeviceToken hashes a raw device token for storage/lookup.
func HashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ClaimDevice exchanges a live pairing code for a device token. The business
// must be APPROVED; suspended businesses cannot onboard players.
func (s *StoreService) ClaimDevice(ctx context.Context, pairingCode string) (*domain.StoreDevice, string, error) {
	code := strings.ToUpper(strings.TrimSpace(pairingCode))
	if code == "" {
		return nil, "", fmt.Errorf("%w: pairingCode", domain.ErrInvalidInput)
	}
	dev, err := s.store.GetDeviceByPairingCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", fmt.Errorf("%w: 만료되었거나 잘못된 페어링 코드", domain.ErrInvalidInput)
		}
		return nil, "", err
	}
	b, err := s.store.GetBusinessByID(ctx, dev.BusinessID)
	if err != nil {
		return nil, "", err
	}
	if b.Status != domain.StoreApproved {
		return nil, "", fmt.Errorf("%w: 승인된 매장만 플레이어를 등록할 수 있습니다", domain.ErrForbidden)
	}
	raw := make([]byte, StoreTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	token := hex.EncodeToString(raw)
	if err := s.store.ActivateDevice(ctx, dev.ID, HashDeviceToken(token)); err != nil {
		return nil, "", err
	}
	claimed, err := s.store.GetDeviceByID(ctx, dev.ID)
	if err != nil {
		return nil, "", err
	}
	return claimed, token, nil
}

// AuthenticateDevice resolves a raw device token to its ACTIVE device.
func (s *StoreService) AuthenticateDevice(ctx context.Context, token string) (*domain.StoreDevice, error) {
	if strings.TrimSpace(token) == "" {
		return nil, domain.ErrInvalidCredentials
	}
	dev, err := s.store.GetDeviceByTokenHash(ctx, HashDeviceToken(strings.TrimSpace(token)))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	return dev, nil
}

// ListDevices returns a business's devices (owner-checked).
func (s *StoreService) ListDevices(ctx context.Context, partnerID, businessID string) ([]*domain.StoreDevice, error) {
	if _, err := s.ownBusiness(ctx, partnerID, businessID); err != nil {
		return nil, err
	}
	return s.store.ListDevicesByBusiness(ctx, businessID)
}

// UpdateDevice renames or re-volumes a device (remote volume, owner-checked).
func (s *StoreService) UpdateDevice(ctx context.Context, partnerID, deviceID, deviceName string, volume int) (*domain.StoreDevice, error) {
	dev, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ownBusiness(ctx, partnerID, dev.BusinessID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(deviceName)
	if name == "" || len([]rune(name)) > MaxDeviceNameChars {
		return nil, fmt.Errorf("%w: 기기명 1..40자", domain.ErrInvalidInput)
	}
	if volume < 0 || volume > 100 {
		return nil, fmt.Errorf("%w: 볼륨 0..100", domain.ErrInvalidInput)
	}
	if err := s.store.UpdateDevice(ctx, deviceID, name, volume); err != nil {
		return nil, err
	}
	return s.store.GetDeviceByID(ctx, deviceID)
}

// RevokeDevice deactivates a device (owner-checked).
func (s *StoreService) RevokeDevice(ctx context.Context, partnerID, deviceID string) error {
	dev, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return err
	}
	if _, err := s.ownBusiness(ctx, partnerID, dev.BusinessID); err != nil {
		return err
	}
	return s.store.RevokeDevice(ctx, deviceID)
}

// UpsertSchedule sets one weekday row (owner-checked, playlist-checked).
func (s *StoreService) UpsertSchedule(ctx context.Context, partnerID, businessID string, weekday int, open, closeTime string, playlistID *string, isClosed bool) (*domain.StoreSchedule, error) {
	b, err := s.ownBusiness(ctx, partnerID, businessID)
	if err != nil {
		return nil, err
	}
	if weekday < 0 || weekday > 6 {
		return nil, fmt.Errorf("%w: weekday 0..6", domain.ErrInvalidInput)
	}
	openT, err := time.Parse("15:04", strings.TrimSpace(open))
	if err != nil {
		return nil, fmt.Errorf("%w: open HH:MM", domain.ErrInvalidInput)
	}
	closeT, err := time.Parse("15:04", strings.TrimSpace(closeTime))
	if err != nil {
		return nil, fmt.Errorf("%w: close HH:MM", domain.ErrInvalidInput)
	}
	if openT.Equal(closeT) {
		return nil, fmt.Errorf("%w: open != close", domain.ErrInvalidInput)
	}
	if playlistID != nil && *playlistID != "" {
		pl, err := s.playlists.GetByID(ctx, *playlistID)
		if err != nil {
			return nil, err
		}
		if !pl.IsPublic && !pl.IsOfficialCuration {
			prof, err := s.partners.GetByID(ctx, b.PartnerID)
			if err != nil {
				return nil, err
			}
			if pl.OwnerUserID != prof.UserID {
				return nil, domain.ErrForbidden
			}
		}
	} else {
		playlistID = nil
	}
	return s.store.UpsertSchedule(ctx, businessID, weekday, openT.Format("15:04"), closeT.Format("15:04"), playlistID, isClosed)
}

// ListSchedules returns a business's weekly schedule (owner-checked).
func (s *StoreService) ListSchedules(ctx context.Context, partnerID, businessID string) ([]*domain.StoreSchedule, error) {
	if _, err := s.ownBusiness(ctx, partnerID, businessID); err != nil {
		return nil, err
	}
	return s.store.ListSchedulesByBusiness(ctx, businessID)
}

// QueueTrack is one playable queue entry.
type QueueTrack struct {
	TrackID    string `json:"trackId"`
	Title      string `json:"title"`
	DurationMs *int   `json:"durationMs,omitempty"`
}

// DeviceQueueResult is the schedule-aware auto queue.
type DeviceQueueResult struct {
	BusinessID   string       `json:"businessId"`
	BusinessName string       `json:"businessName"`
	Open         bool         `json:"open"`
	Reason       string       `json:"reason"`
	PlaylistID   *string      `json:"playlistId,omitempty"`
	PlaylistName string       `json:"playlistName,omitempty"`
	Volume       int          `json:"volume"`
	Tracks       []QueueTrack `json:"tracks"`
}

// DeviceQueue builds the current auto queue for an ACTIVE device.
func (s *StoreService) DeviceQueue(ctx context.Context, deviceID string) (*DeviceQueueResult, error) {
	dev, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if dev.Status != domain.DeviceActive {
		return nil, domain.ErrInvalidCredentials
	}
	b, err := s.store.GetBusinessByID(ctx, dev.BusinessID)
	if err != nil {
		return nil, err
	}
	out := &DeviceQueueResult{
		BusinessID: b.ID, BusinessName: b.StoreName,
		Volume: dev.Volume, Tracks: []QueueTrack{},
	}
	if b.Status != domain.StoreApproved {
		out.Reason = "business_" + strings.ToLower(b.Status)
		return out, nil
	}
	now := s.clock()
	schedules, err := s.store.ListSchedulesByBusiness(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	var today *domain.StoreSchedule
	for _, sc := range schedules {
		if sc.Weekday == int(now.Weekday()) {
			today = sc
			break
		}
	}
	playlistID := ""
	playlistName := ""
	switch {
	case today == nil:
		// No row for today: fall back to official curation all day.
		official, err := s.playlists.ListOfficialCuration(ctx)
		if err != nil || len(official) == 0 {
			out.Reason = "no_schedule_no_curation"
			return out, nil
		}
		playlistID, playlistName = official[0].ID, official[0].Title
		out.Open = true
		out.Reason = "fallback_curation"
	case today.IsClosed:
		out.Reason = "closed_day"
		return out, nil
	default:
		if !withinHours(now, today.OpenTime, today.CloseTime) {
			out.Reason = "off_hours"
			return out, nil
		}
		out.Open = true
		if today.PlaylistID != nil && *today.PlaylistID != "" {
			playlistID = *today.PlaylistID
			if pl, err := s.playlists.GetByID(ctx, playlistID); err == nil {
				playlistName = pl.Title
			}
			out.Reason = "scheduled"
		} else {
			official, err := s.playlists.ListOfficialCuration(ctx)
			if err != nil || len(official) == 0 {
				out.Reason = "no_playlist_no_curation"
				return out, nil
			}
			playlistID, playlistName = official[0].ID, official[0].Title
			out.Reason = "fallback_curation"
		}
	}
	out.PlaylistID = &playlistID
	out.PlaylistName = playlistName
	entries, err := s.playlists.ListTracks(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		track, err := s.tracks.GetByID(ctx, e.TrackID)
		if err != nil || track.Status != domain.TrackReady || track.HLSPrefix == nil {
			continue
		}
		out.Tracks = append(out.Tracks, QueueTrack{TrackID: track.ID, Title: track.Title, DurationMs: track.DurationMs})
	}
	return out, nil
}

// withinHours reports whether now falls in [open, close), overnight-aware.
func withinHours(now time.Time, open, close string) bool {
	openT, err := time.Parse("15:04", open)
	if err != nil {
		return false
	}
	closeT, err := time.Parse("15:04", close)
	if err != nil {
		return false
	}
	minutes := now.Hour()*60 + now.Minute()
	o := openT.Hour()*60 + openT.Minute()
	c := closeT.Hour()*60 + closeT.Minute()
	if c > o {
		return minutes >= o && minutes < c
	}
	return minutes >= o || minutes < c
}

// Heartbeat records a device play; 30s+ counts like listener plays.
func (s *StoreService) Heartbeat(ctx context.Context, deviceID, trackID string, durationSecs int) (bool, error) {
	dev, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return false, err
	}
	if dev.Status != domain.DeviceActive {
		return false, domain.ErrInvalidCredentials
	}
	b, err := s.store.GetBusinessByID(ctx, dev.BusinessID)
	if err != nil {
		return false, err
	}
	if b.Status != domain.StoreApproved {
		return false, fmt.Errorf("%w: 매장 승인 후 집계됩니다", domain.ErrForbidden)
	}
	track, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		return false, err
	}
	if track.Status != domain.TrackReady {
		return false, domain.ErrNotStreamable
	}
	if durationSecs < 0 {
		durationSecs = 0
	}
	counted := durationSecs >= StoreCountSeconds
	if _, err := s.store.AppendPlayLog(ctx, deviceID, trackID, durationSecs, counted); err != nil {
		return false, err
	}
	_ = s.store.TouchDeviceSeen(ctx, deviceID)
	return counted, nil
}

// DeviceLogs returns recent plays for a device (owner-checked).
func (s *StoreService) DeviceLogs(ctx context.Context, partnerID, deviceID string, limit int) ([]*domain.StorePlayLog, error) {
	dev, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ownBusiness(ctx, partnerID, dev.BusinessID); err != nil {
		return nil, err
	}
	return s.store.ListPlayLogsByDevice(ctx, deviceID, limit)
}
