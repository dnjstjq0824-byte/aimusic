package service

import (
	"context"
	"errors"
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
//
// Counting is cumulative: clients send small playMs deltas (~10s cadence), so
// the delta is added to the uncounted play_ms accumulated since the last
// counted play for this user+track, and the heartbeat that crosses the 30s
// threshold counts exactly once.
func (s *PlaybackService) Heartbeat(ctx context.Context, in HeartbeatInput) (*HeartbeatOutput, error) {
	if in.PlayMs < 0 {
		return nil, fmt.Errorf("%w: playMs", domain.ErrInvalidInput)
	}
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
	prior, err := s.repos.Streaming.SumPlayMsSinceLastCounted(ctx, in.UserID, in.TrackID)
	if err != nil {
		return nil, err
	}
	counted := !fraud && prior+in.PlayMs >= ValidStreamMs
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

// Withholding per mille: business income 3.3% = 33/1000 (default),
// occasional other income 8.8% = 88/1000 (B9).
const (
	withholdingNumBusiness = 33
	withholdingNumOther    = 88
	withholdingDen         = 1000
)

// Partner share of attributed distribution revenue: 70/100 (방향결정서 §5-1).
const (
	partnerShareNum = 70
	partnerShareDen = 100
)

// SettlementService calculates partner settlements.
type SettlementService struct {
	repos  *repository.Repos
	cipher *apcrypto.Cipher
}

// SettlementInput carries the admin-attested revenue inputs for one
// settlement calculation. No in-system revenue ledger exists for the paid
// channels yet, so the period totals are attested from external books
// (store subscriptions, license/direct deals) and recorded on the row.
type SettlementInput struct {
	// StorePoolKRW is the total store-subscription pool collected for the
	// period, split across partners by qualified store play time.
	StorePoolKRW int64
	// DirectKRW is this partner's directly attributed revenue (direct
	// sales and license deals on their tracks).
	DirectKRW  int64
	IncomeType string
}

// withholdNum validates the input and resolves the withholding per-mille
// numerator (BUSINESS_33 default, OTHER_88 occasional income).
func (in SettlementInput) withholdNum() (int64, error) {
	if in.StorePoolKRW < 0 || in.DirectKRW < 0 {
		return 0, fmt.Errorf("%w: storePoolKrw/directKrw must be >= 0", domain.ErrInvalidInput)
	}
	incomeType := in.IncomeType
	if incomeType == "" {
		incomeType = domain.IncomeBusiness33
	}
	if !domain.ValidIncomeType(incomeType) {
		return 0, fmt.Errorf("%w: incomeType must be BUSINESS_33|OTHER_88", domain.ErrInvalidInput)
	}
	if incomeType == domain.IncomeOther88 {
		return withholdingNumOther, nil
	}
	return withholdingNumBusiness, nil
}

// splitSettlement7030 applies the 70:30 canon in pure integer math,
// flooring at each step. The pool share is capped at the pool so skew
// between the partner and total basis reads can never overpay.
func splitSettlement7030(storePool, direct, partnerSecs, totalSecs, withholdNum int64) (poolShare, attributed, gross, withholding, net int64) {
	if totalSecs > 0 && partnerSecs > 0 && storePool > 0 {
		poolShare = storePool * partnerSecs / totalSecs
		if poolShare > storePool {
			poolShare = storePool
		}
	}
	attributed = poolShare + direct
	gross = attributed * partnerShareNum / partnerShareDen
	withholding = gross * withholdNum / withholdingDen
	net = gross - withholding
	return poolShare, attributed, gross, withholding, net
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

// Calculate settles one partner period under the 70:30 canon (방향결정서
// §5-1): the store pool splits by qualified 30s+ store play time, direct
// revenue attributes in full, the partner keeps 70%, withholding applies,
// and violations forfeit (FORFEITED_VIOLATION, net zero). General streaming
// plays never create money. Business partners settle via tax invoice
// outside this flow (B9).
func (s *SettlementService) Calculate(ctx context.Context, partnerID string, start, end time.Time, in SettlementInput) (*domain.PartnerSettlement, error) {
	if partnerID == "" || !start.Before(end) {
		return nil, fmt.Errorf("%w: partner/period", domain.ErrInvalidInput)
	}
	num, err := in.withholdNum()
	if err != nil {
		return nil, err
	}
	incomeType := in.IncomeType
	if incomeType == "" {
		incomeType = domain.IncomeBusiness33
	}
	partnerSecs, err := s.repos.Store.SumQualifiedStoreSecs(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	totalSecs, err := s.repos.Store.SumQualifiedStoreSecsAll(ctx, start, end)
	if err != nil {
		return nil, err
	}
	playCount, err := s.repos.Store.CountPartnerPlaysInWindow(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	_, _, gross, withholding, net := splitSettlement7030(in.StorePoolKRW, in.DirectKRW, partnerSecs, totalSecs, num)
	settlement := &domain.PartnerSettlement{
		PartnerID:          partnerID,
		PeriodStart:        start,
		PeriodEnd:          end,
		GrossAmountKRW:     gross,
		WithholdingKRW:     withholding,
		NetAmountKRW:       net,
		PlayCount:          playCount,
		StorePoolKRW:       in.StorePoolKRW,
		DirectKRW:          in.DirectKRW,
		StorePlaySecs:      partnerSecs,
		TotalStorePlaySecs: totalSecs,
		Status:             domain.SettlePending,
		IncomeType:         incomeType,
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

// canHold reports whether a settlement in status may be held (P2-2 #12).
func canHold(status string) bool { return status == domain.SettlePending }

// canRelease reports whether a settlement in status may be released.
func canRelease(status string) bool { return status == domain.SettleHeld }

// Approve moves a settlement to APPROVED (never from FORFEITED or HELD).
func (s *SettlementService) Approve(ctx context.Context, partnerID string, start, end time.Time) (*domain.PartnerSettlement, error) {
	cur, err := s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	if cur.Status == domain.SettleForfeited {
		return nil, fmt.Errorf("%w: forfeited settlement", domain.ErrInvalidInput)
	}
	if cur.Status == domain.SettleHeld {
		return nil, fmt.Errorf("%w: held settlement must be released first", domain.ErrInvalidInput)
	}
	if err := s.repos.Settlements.UpdateStatus(ctx, cur.ID, domain.SettleApproved, nil); err != nil {
		return nil, err
	}
	return s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
}

// Hold suspends a PENDING settlement while confirmed fraud is reviewed.
// Reason is required evidence; the actor is the holding admin.
func (s *SettlementService) Hold(ctx context.Context, partnerID string, start, end time.Time, reason, actorID string) (*domain.PartnerSettlement, error) {
	if partnerID == "" || !start.Before(end) {
		return nil, fmt.Errorf("%w: partner/period", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: hold reason", domain.ErrInvalidInput)
	}
	cur, err := s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	if !canHold(cur.Status) {
		return nil, fmt.Errorf("%w: only PENDING settlements can be held", domain.ErrInvalidInput)
	}
	if err := s.repos.Settlements.Hold(ctx, cur.ID, strings.TrimSpace(reason), actorID); err != nil {
		return nil, err
	}
	return s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
}

// Release returns a HELD settlement to PENDING for the normal flow.
func (s *SettlementService) Release(ctx context.Context, partnerID string, start, end time.Time) (*domain.PartnerSettlement, error) {
	if partnerID == "" || !start.Before(end) {
		return nil, fmt.Errorf("%w: partner/period", domain.ErrInvalidInput)
	}
	cur, err := s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
	if err != nil {
		return nil, err
	}
	if !canRelease(cur.Status) {
		return nil, fmt.Errorf("%w: only HELD settlements can be released", domain.ErrInvalidInput)
	}
	if err := s.repos.Settlements.Release(ctx, cur.ID); err != nil {
		return nil, err
	}
	return s.repos.Settlements.GetByPartnerPeriod(ctx, partnerID, start, end)
}

// MarkPaid records an executed bank transfer (APPROVED -> PAID exactly once).
// The transfer itself is manual; reference carries the bank slip number.
func (s *SettlementService) MarkPaid(ctx context.Context, partnerID string, start, end time.Time, reference string) (*domain.PartnerSettlement, error) {
	if partnerID == "" || !start.Before(end) {
		return nil, fmt.Errorf("%w: partner/period", domain.ErrInvalidInput)
	}
	settlement, err := s.repos.Settlements.MarkPaid(ctx, partnerID, start, end, strings.TrimSpace(reference))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: only APPROVED settlements can be marked paid", domain.ErrInvalidInput)
		}
		return nil, err
	}
	return settlement, nil
}
