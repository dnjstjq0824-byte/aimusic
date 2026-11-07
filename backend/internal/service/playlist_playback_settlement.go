package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
	apcrypto "avibe-platform/backend/internal/pkg/crypto"
	"avibe-platform/backend/internal/repository"
)

// PlaylistService handles playlist CRUD with ownership checks.
type PlaylistService struct {
	repos *repository.Repos
}

// Create makes a playlist owned by userID.
func (s *PlaylistService) Create(ctx context.Context, ownerID, title string, isPublic bool) (*domain.Playlist, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: title", domain.ErrInvalidInput)
	}
	return s.repos.Playlists.Create(ctx, ownerID, title, isPublic)
}

// Get returns a playlist visible to caller (owner or public).
func (s *PlaylistService) Get(ctx context.Context, callerID, playlistID string) (*domain.Playlist, error) {
	p, err := s.repos.Playlists.GetByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if p.OwnerUserID != callerID && !p.IsPublic {
		return nil, domain.ErrForbidden
	}
	return p, nil
}

// ListMine returns the caller's playlists.
func (s *PlaylistService) ListMine(ctx context.Context, callerID string) ([]*domain.Playlist, error) {
	return s.repos.Playlists.ListByOwner(ctx, callerID)
}

// Update renames a playlist; owner only.
func (s *PlaylistService) Update(ctx context.Context, callerID, playlistID, title string, isPublic bool) (*domain.Playlist, error) {
	if err := s.requireOwner(ctx, callerID, playlistID); err != nil {
		return nil, err
	}
	if err := s.repos.Playlists.Update(ctx, playlistID, title, isPublic); err != nil {
		return nil, err
	}
	return s.repos.Playlists.GetByID(ctx, playlistID)
}

// Delete removes a playlist; owner only.
func (s *PlaylistService) Delete(ctx context.Context, callerID, playlistID string) error {
	if err := s.requireOwner(ctx, callerID, playlistID); err != nil {
		return err
	}
	return s.repos.Playlists.Delete(ctx, playlistID)
}

// AddTrack appends a READY track; owner only.
func (s *PlaylistService) AddTrack(ctx context.Context, callerID, playlistID, trackID string) error {
	if err := s.requireOwner(ctx, callerID, playlistID); err != nil {
		return err
	}
	track, err := s.repos.Tracks.GetByID(ctx, trackID)
	if err != nil {
		return err
	}
	if track.Status != domain.TrackReady {
		return domain.ErrNotStreamable
	}
	return s.repos.Playlists.AddTrack(ctx, playlistID, trackID)
}

// RemoveTrack removes an entry; owner only.
func (s *PlaylistService) RemoveTrack(ctx context.Context, callerID, playlistID, trackID string) error {
	if err := s.requireOwner(ctx, callerID, playlistID); err != nil {
		return err
	}
	return s.repos.Playlists.RemoveTrack(ctx, playlistID, trackID)
}

// Reorder rewrites entry order; owner only. All IDs must already belong.
func (s *PlaylistService) Reorder(ctx context.Context, callerID, playlistID string, trackIDs []string) error {
	if err := s.requireOwner(ctx, callerID, playlistID); err != nil {
		return err
	}
	current, err := s.repos.Playlists.ListTracks(ctx, playlistID)
	if err != nil {
		return err
	}
	if len(current) != len(trackIDs) {
		return fmt.Errorf("%w: reorder must include every track", domain.ErrInvalidInput)
	}
	known := map[string]bool{}
	for _, e := range current {
		known[e.TrackID] = true
	}
	for _, id := range trackIDs {
		if !known[id] {
			return fmt.Errorf("%w: unknown track in reorder", domain.ErrInvalidInput)
		}
	}
	return s.repos.Playlists.Reorder(ctx, playlistID, trackIDs)
}

// ListTracks returns ordered entries visible to caller.
func (s *PlaylistService) ListTracks(ctx context.Context, callerID, playlistID string) ([]*domain.PlaylistTrack, error) {
	if _, err := s.Get(ctx, callerID, playlistID); err != nil {
		return nil, err
	}
	return s.repos.Playlists.ListTracks(ctx, playlistID)
}

func (s *PlaylistService) requireOwner(ctx context.Context, callerID, playlistID string) error {
	p, err := s.repos.Playlists.GetByID(ctx, playlistID)
	if err != nil {
		return err
	}
	if p.OwnerUserID != callerID {
		return domain.ErrForbidden
	}
	return nil
}

// ValidStreamMs is the playback duration that counts as a valid stream.
const ValidStreamMs = 30000

const (
	maxPlaysPerHour      = 120
	maxTrackPlaysPerHour = 40
)

// HeartbeatInput is one playback progress report.
type HeartbeatInput struct {
	UserID     string
	TrackID    string
	PositionMs int
	PlayMs     int
	JTI        string
	IP         string
	UA         string
}

// HeartbeatOutput reports counting and fraud decisions.
type HeartbeatOutput struct {
	CountedAsPlay  bool `json:"countedAsPlay"`
	FraudSuspected bool `json:"fraudSuspected"`
}

// PlaybackService handles heartbeats with 30s valid-stream and fraud flags.
type PlaybackService struct {
	repos *repository.Repos
}

// Heartbeat logs playback, counts 30s+ plays, and flags fraud windows.
func (s *PlaybackService) Heartbeat(ctx context.Context, in HeartbeatInput) (*HeartbeatOutput, error) {
	track, err := s.repos.Tracks.GetByID(ctx, in.TrackID)
	if err != nil {
		return nil, err
	}
	if track.Status != domain.TrackReady {
		return nil, domain.ErrNotStreamable
	}
	since := time.Now().Add(-time.Hour)
	total, err := s.repos.Streaming.CountUserPlaysSince(ctx, in.UserID, since)
	if err != nil {
		return nil, err
	}
	perTrack, err := s.repos.Streaming.CountUserTrackPlaysSince(ctx, in.UserID, in.TrackID, since)
	if err != nil {
		return nil, err
	}
	fraud := total >= maxPlaysPerHour || perTrack >= maxTrackPlaysPerHour
	counted := in.PlayMs >= ValidStreamMs && !fraud
	var jti *string
	if in.JTI != "" {
		jti = &in.JTI
	}
	var ip, ua *string
	if in.IP != "" {
		ip = &in.IP
	}
	if in.UA != "" {
		ua = &in.UA
	}
	if err := s.repos.Streaming.Insert(ctx, &domain.StreamingLog{
		UserID: &in.UserID, TrackID: &in.TrackID, PartnerID: &track.PartnerID,
		PlayMs: in.PlayMs, CountedAsPlay: counted, IP: ip, UserAgent: ua, HLSTokenJTI: jti,
	}); err != nil {
		return nil, err
	}
	if counted {
		if err := s.repos.Tracks.IncrementPlayCount(ctx, in.TrackID); err != nil {
			return nil, err
		}
	}
	if in.PositionMs >= 0 {
		if _, err := s.repos.Playback.UpsertState(ctx, in.UserID, in.TrackID, in.PositionMs); err != nil {
			return nil, err
		}
	}
	return &HeartbeatOutput{CountedAsPlay: counted, FraudSuspected: fraud}, nil
}

// GetState returns the resume position.
func (s *PlaybackService) GetState(ctx context.Context, userID, trackID string) (*domain.UserPlaybackState, error) {
	return s.repos.Playback.GetState(ctx, userID, trackID)
}

// SetPosition stores the resume position.
func (s *PlaybackService) SetPosition(ctx context.Context, userID, trackID string, positionMs int) (*domain.UserPlaybackState, error) {
	if positionMs < 0 {
		return nil, fmt.Errorf("%w: positionMs", domain.ErrInvalidInput)
	}
	return s.repos.Playback.UpsertState(ctx, userID, trackID, positionMs)
}

// Withholding per mille: 3.3% = 33/1000.
const (
	withholdingNum = 33
	withholdingDen = 1000
)

// SettlementService calculates partner settlements.
type SettlementService struct {
	repos                 *repository.Repos
	defaultRatePerPlayKRW int64
	cipher                *apcrypto.Cipher
}

// UpdateSettlementAccount stores bank + AES-256-GCM encrypted account number.
// It fails closed when no cipher is configured: plaintext is never stored.
func (s *SettlementService) UpdateSettlementAccount(ctx context.Context, partnerID, bank, accountNumber string) error {
	if partnerID == "" || strings.TrimSpace(bank) == "" || strings.TrimSpace(accountNumber) == "" {
		return fmt.Errorf("%w: partner/bank/accountNumber", domain.ErrInvalidInput)
	}
	if s.cipher == nil {
		return fmt.Errorf("%w: settlement encryption unavailable", domain.ErrInvalidInput)
	}
	enc, err := s.cipher.Encrypt([]byte(strings.TrimSpace(accountNumber)))
	if err != nil {
		return err
	}
	return s.repos.Partners.UpdateSettlement(ctx, partnerID, strings.TrimSpace(bank), enc)
}

// DecryptSettlementAccount opens an AES-256-GCM account value for payout use.
// Callers must never log or return the plaintext; prefer SettlementAccountPreview.
func (s *SettlementService) DecryptSettlementAccount(enc string) (string, error) {
	if s.cipher == nil {
		return "", fmt.Errorf("%w: settlement encryption unavailable", domain.ErrInvalidInput)
	}
	plain, err := s.cipher.Decrypt(enc)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// SettlementAccountPreview returns the bank name and a masked account preview
// (all but the last 4 digits masked) for admin review screens.
func (s *SettlementService) SettlementAccountPreview(ctx context.Context, partnerID string) (bank, masked string, err error) {
	p, err := s.repos.Partners.GetByID(ctx, partnerID)
	if err != nil {
		return "", "", err
	}
	if p.SettlementBank != nil {
		bank = *p.SettlementBank
	}
	if p.SettlementAccountEncrypted == nil || *p.SettlementAccountEncrypted == "" {
		return bank, "", nil
	}
	plain, err := s.DecryptSettlementAccount(*p.SettlementAccountEncrypted)
	if err != nil {
		return "", "", err
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, plain)
	if len(digits) <= 4 {
		return bank, "****", nil
	}
	return bank, strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:], nil
}

// Calculate counts valid plays in [start, end), applies 3.3% withholding, and
// forfeits (FORFEITED_VIOLATION, net zero) when a violation is active.
func (s *SettlementService) Calculate(ctx context.Context, partnerID string, start, end time.Time, ratePerPlayKRW int64) (*domain.PartnerSettlement, error) {
	if partnerID == "" || !start.Before(end) {
		return nil, fmt.Errorf("%w: partner/period", domain.ErrInvalidInput)
	}
	if ratePerPlayKRW <= 0 {
		ratePerPlayKRW = s.defaultRatePerPlayKRW
	}
	plays, err := s.repos.Streaming.CountPartnerPlaysInWindow(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	gross := plays * ratePerPlayKRW
	withholding := gross * withholdingNum / withholdingDen
	net := gross - withholding
	settlement := &domain.PartnerSettlement{
		PartnerID:      partnerID,
		PeriodStart:    start,
		PeriodEnd:      end,
		GrossAmountKRW: gross,
		WithholdingKRW: withholding,
		NetAmountKRW:   net,
		PlayCount:      plays,
		Status:         domain.SettlePending,
	}
	violated, err := s.ViolationActive(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	if violated {
		reason := domain.ForfeitViolation
		settlement.Status = domain.SettleForfeited
		settlement.ForfeitureReason = &reason
		settlement.NetAmountKRW = 0
	}
	return s.repos.Settlements.Upsert(ctx, settlement)
}

// ViolationActive reports whether the partner is suspended/terminated or has
// upheld takedowns on record.
func (s *SettlementService) ViolationActive(ctx context.Context, partnerID string) (bool, error) {
	profile, err := s.repos.Partners.GetByID(ctx, partnerID)
	if err != nil {
		return false, err
	}
	if profile.Status == domain.PartnerSuspended || profile.Status == domain.PartnerTerminated {
		return true, nil
	}
	upheld, err := s.repos.Disputes.CountUpheldByPartner(ctx, partnerID)
	if err != nil {
		return false, err
	}
	return upheld > 0, nil
}

// Approve moves a settlement to APPROVED (never from FORFEITED).
func (s *SettlementService) Approve(ctx context.Context, partnerID string, start, end time.Time) (*domain.PartnerSettlement, error) {
	cur, err := s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	if cur.Status == domain.SettleForfeited {
		return nil, fmt.Errorf("%w: forfeited settlement", domain.ErrInvalidInput)
	}
	if err := s.repos.Settlements.UpdateStatus(ctx, cur.ID, domain.SettleApproved, nil); err != nil {
		return nil, err
	}
	return s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
}
