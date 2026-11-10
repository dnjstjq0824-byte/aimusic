package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// LegalRepo accesses legal_terms and user_term_consents.
type LegalRepo struct{ db *sql.DB }

// ListTerms returns all legal terms ordered by code and version.
func (r *LegalRepo) ListTerms(ctx context.Context) ([]*domain.LegalTerm, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, code, version, title_ko, body_md, is_required, effective_at, created_at
		FROM legal_terms ORDER BY code ASC, version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.LegalTerm
	for rows.Next() {
		t := &domain.LegalTerm{}
		if err := rows.Scan(&t.ID, &t.Code, &t.Version, &t.TitleKo, &t.BodyMd, &t.IsRequired, &t.EffectiveAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTermByCode returns the latest effective term for a code.
func (r *LegalRepo) GetTermByCode(ctx context.Context, code string) (*domain.LegalTerm, error) {
	t := &domain.LegalTerm{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text, code, version, title_ko, body_md, is_required, effective_at, created_at
		FROM legal_terms WHERE code = $1 ORDER BY effective_at DESC LIMIT 1`, code,
	).Scan(&t.ID, &t.Code, &t.Version, &t.TitleKo, &t.BodyMd, &t.IsRequired, &t.EffectiveAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return t, err
}

// CreateConsent records a user consent (idempotent per user+terms).
func (r *LegalRepo) CreateConsent(ctx context.Context, userID, termsID string, ip, ua *string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_term_consents (user_id, terms_id, ip, user_agent)
		VALUES ($1::uuid, $2::uuid, $3, $4)
		ON CONFLICT (user_id, terms_id) DO NOTHING`,
		userID, termsID, nullStr(ip), nullStr(ua))
	return err
}

// ListPendingConsents returns the latest-effective required terms per code
// the user has not consented to (new required versions re-appear here).
func (r *LegalRepo) ListPendingConsents(ctx context.Context, userID string) ([]*domain.LegalTerm, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, code, version, title_ko, body_md, is_required, effective_at, created_at
		FROM legal_terms t
		WHERE t.is_required
		  AND t.effective_at = (SELECT MAX(effective_at) FROM legal_terms WHERE code = t.code)
		  AND NOT EXISTS (
			SELECT 1 FROM user_term_consents c
			WHERE c.user_id = $1::uuid AND c.terms_id = t.id)
		ORDER BY t.code ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.LegalTerm
	for rows.Next() {
		t := &domain.LegalTerm{}
		if err := rows.Scan(&t.ID, &t.Code, &t.Version, &t.TitleKo, &t.BodyMd, &t.IsRequired, &t.EffectiveAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListConsentsByUser returns a user's consent records.
func (r *LegalRepo) ListConsentsByUser(ctx context.Context, userID string) ([]*domain.UserTermConsent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, terms_id::text, consented_at, ip, user_agent
		FROM user_term_consents WHERE user_id = $1::uuid ORDER BY consented_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.UserTermConsent
	for rows.Next() {
		c := &domain.UserTermConsent{}
		var ip, ua sql.NullString
		if err := rows.Scan(&c.ID, &c.UserID, &c.TermsID, &c.ConsentedAt, &ip, &ua); err != nil {
			return nil, err
		}
		c.IP = strPtr(ip)
		c.UserAgent = strPtr(ua)
		out = append(out, c)
	}
	return out, rows.Err()
}

// DisputeRepo accesses copyright_disputes.
type DisputeRepo struct{ db *sql.DB }

func scanDispute(row interface{ Scan(...any) error }) (*domain.CopyrightDispute, error) {
	d := &domain.CopyrightDispute{}
	var reporter, resolver sql.NullString
	var resolvedAt, takedownReq, partnerNotif, resumeReq, resumeSched, resumed, slaNotif sql.NullTime
	err := row.Scan(&d.ID, &d.TrackID, &reporter, &d.Reason, &d.Description, &d.Status,
		&d.StrikeApplied, &resolver, &resolvedAt,
		&takedownReq, &partnerNotif, &resumeReq, &resumeSched, &resumed, &slaNotif,
		&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.ReporterUserID = strPtr(reporter)
	d.ResolvedBy = strPtr(resolver)
	d.ResolvedAt = timePtr(resolvedAt)
	d.TakedownRequestedAt = timePtr(takedownReq)
	d.PartnerNotifiedAt = timePtr(partnerNotif)
	d.ResumeRequestedAt = timePtr(resumeReq)
	d.ResumeScheduledAt = timePtr(resumeSched)
	d.ResumedAt = timePtr(resumed)
	d.SLANotifiedAt = timePtr(slaNotif)
	return d, nil
}

const disputeColumns = `id::text, track_id::text, reporter_user_id::text, reason, description,
	status, strike_applied, resolved_by::text, resolved_at,
	takedown_requested_at, partner_notified_at, resume_requested_at, resume_scheduled_at, resumed_at, sla_notified_at,
	created_at, updated_at`

func (r *DisputeRepo) Create(ctx context.Context, trackID string, reporterID *string, reason, desc string) (*domain.CopyrightDispute, error) {
	d, err := scanDispute(r.db.QueryRowContext(ctx, `
		INSERT INTO copyright_disputes (track_id, reporter_user_id, reason, description, takedown_requested_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, NOW())
		RETURNING `+disputeColumns, trackID, nullStr(reporterID), reason, desc))
	return d, err
}

func (r *DisputeRepo) GetByID(ctx context.Context, id string) (*domain.CopyrightDispute, error) {
	d, err := scanDispute(r.db.QueryRowContext(ctx,
		`SELECT `+disputeColumns+` FROM copyright_disputes WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return d, err
}

// ListByStatus returns disputes filtered by status.
func (r *DisputeRepo) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*domain.CopyrightDispute, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+disputeColumns+`
		FROM copyright_disputes WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		status, clampLimit(limit, 20, 100), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.CopyrightDispute
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListOpen returns OPEN and UNDER_REVIEW disputes for the admin queue.
func (r *DisputeRepo) ListOpen(ctx context.Context, limit, offset int) ([]*domain.CopyrightDispute, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+disputeColumns+`
		FROM copyright_disputes WHERE status IN ('OPEN','UNDER_REVIEW')
		ORDER BY created_at ASC LIMIT $1 OFFSET $2`, clampLimit(limit, 20, 100), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.CopyrightDispute
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// AppendCounter records a partner counter-claim and reopens review.
// It stamps the §103 resume request and schedules review 7 days out.
func (r *DisputeRepo) AppendCounter(ctx context.Context, id, note string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE copyright_disputes
		SET description = description || $2, status = 'UNDER_REVIEW',
			resume_requested_at = NOW(), resume_scheduled_at = NOW() + INTERVAL '7 days'
		WHERE id = $1::uuid`, id, "\n[COUNTER] "+note)
	return err
}

// Resolve records an admin resolution decision.
// Dismissals stamp resumed_at (the §103 resume); upholds stamp the partner
// decision notice when it has not been recorded yet.
func (r *DisputeRepo) Resolve(ctx context.Context, id, status, resolverID string, strikeApplied bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE copyright_disputes
		SET status = $2, strike_applied = $3, resolved_by = $4::uuid, resolved_at = NOW(),
			resumed_at = CASE WHEN $2 = 'DISMISSED' THEN NOW() ELSE resumed_at END,
			partner_notified_at = CASE WHEN $2 = 'UPHELD_TAKEDOWN' AND partner_notified_at IS NULL THEN NOW() ELSE partner_notified_at END
		WHERE id = $1::uuid`, id, status, strikeApplied, resolverID)
	return err
}

// StampPartnerNotified records the §103 takedown notice on every open dispute
// for a track (called when an admin takes the track down).
func (r *DisputeRepo) StampPartnerNotified(ctx context.Context, trackID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE copyright_disputes
		SET partner_notified_at = NOW()
		WHERE track_id = $1::uuid AND status IN ('OPEN','UNDER_REVIEW') AND partner_notified_at IS NULL`,
		trackID)
	return err
}

// ListSLABreached returns OPEN/UNDER_REVIEW disputes older than slaHours
// that have not been SLA-notified yet (oldest first, capped for one sweep).
func (r *DisputeRepo) ListSLABreached(ctx context.Context, slaHours int, limit int) ([]*domain.CopyrightDispute, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+disputeColumns+`
		FROM copyright_disputes
		WHERE status IN ('OPEN','UNDER_REVIEW')
		  AND sla_notified_at IS NULL
		  AND created_at < NOW() - make_interval(hours => $1)
		ORDER BY created_at ASC LIMIT $2`, slaHours, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.CopyrightDispute{}
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// StampSLANotified marks a dispute as SLA-notified (exactly once: only rows
// still un-notified are stamped, so concurrent sweepers cannot double-alert).
func (r *DisputeRepo) StampSLANotified(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE copyright_disputes SET sla_notified_at = NOW()
		WHERE id = $1::uuid AND sla_notified_at IS NULL`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// ListByReporter returns disputes filed by a user (data export).
func (r *DisputeRepo) ListByReporter(ctx context.Context, reporterID string, limit int) ([]*domain.CopyrightDispute, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+disputeColumns+` FROM copyright_disputes
		WHERE reporter_user_id = $1::uuid ORDER BY created_at DESC LIMIT $2`, reporterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.CopyrightDispute{}
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CountOpen returns the open/under-review dispute count.
func (r *DisputeRepo) CountOpen(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM copyright_disputes WHERE status IN ('OPEN','UNDER_REVIEW')`).Scan(&n)
	return n, err
}

// CountUpheldByPartner counts takedown-upheld disputes on a partner's tracks.
func (r *DisputeRepo) CountUpheldByPartner(ctx context.Context, partnerID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM copyright_disputes d
		JOIN tracks t ON t.id = d.track_id
		WHERE t.partner_id = $1::uuid AND d.status = 'UPHELD_TAKEDOWN'`, partnerID).Scan(&n)
	return n, err
}

// BlockRepo accesses user_blocks.
type BlockRepo struct{ db *sql.DB }

func (r *BlockRepo) Create(ctx context.Context, userID string, blockedBy *string, reason string, strike int, expiresAt *time.Time) (*domain.UserBlock, error) {
	b := &domain.UserBlock{}
	var by sql.NullString
	var exp sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO user_blocks (user_id, blocked_by, reason, strike_number, expires_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		RETURNING id::text, user_id::text, blocked_by::text, reason, strike_number, expires_at, created_at`,
		userID, nullStr(blockedBy), reason, strike, nullTime(expiresAt),
	).Scan(&b.ID, &b.UserID, &by, &b.Reason, &b.StrikeNumber, &exp, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	b.BlockedBy = strPtr(by)
	b.ExpiresAt = timePtr(exp)
	return b, nil
}

// ListByUser returns blocks for a user, newest first.
func (r *BlockRepo) ListByUser(ctx context.Context, userID string) ([]*domain.UserBlock, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, blocked_by::text, reason, strike_number, expires_at, created_at
		FROM user_blocks WHERE user_id = $1::uuid ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.UserBlock
	for rows.Next() {
		b := &domain.UserBlock{}
		var by sql.NullString
		var exp sql.NullTime
		if err := rows.Scan(&b.ID, &b.UserID, &by, &b.Reason, &b.StrikeNumber, &exp, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.BlockedBy = strPtr(by)
		b.ExpiresAt = timePtr(exp)
		out = append(out, b)
	}
	return out, rows.Err()
}

// CountStrikes returns the number of strikes on record for a user.
func (r *BlockRepo) CountStrikes(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_blocks WHERE user_id = $1::uuid`, userID).Scan(&n)
	return n, err
}

// BlockArtist records a user blocking a partner/artist (App Store / Google Play UGC rule).
func (r *BlockRepo) BlockArtist(ctx context.Context, blockerUserID, blockedPartnerID, reason string) (*domain.UserArtistBlock, error) {
	b := &domain.UserArtistBlock{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO user_artist_blocks (blocker_user_id, blocked_partner_id, reason)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (blocker_user_id, blocked_partner_id) DO UPDATE SET reason = EXCLUDED.reason
		RETURNING id::text, blocker_user_id::text, blocked_partner_id::text, reason, created_at`,
		blockerUserID, blockedPartnerID, reason,
	).Scan(&b.ID, &b.BlockerUserID, &b.BlockedPartnerID, &b.Reason, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UnblockArtist removes a user's block against a partner/artist.
func (r *BlockRepo) UnblockArtist(ctx context.Context, blockerUserID, blockedPartnerID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM user_artist_blocks WHERE blocker_user_id = $1::uuid AND blocked_partner_id = $2::uuid`,
		blockerUserID, blockedPartnerID)
	return err
}

// ListBlockedArtists returns the partner blocks created by a user.
func (r *BlockRepo) ListBlockedArtists(ctx context.Context, blockerUserID string) ([]*domain.UserArtistBlock, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, blocker_user_id::text, blocked_partner_id::text, reason, created_at
		FROM user_artist_blocks WHERE blocker_user_id = $1::uuid ORDER BY created_at DESC`, blockerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.UserArtistBlock
	for rows.Next() {
		b := &domain.UserArtistBlock{}
		if err := rows.Scan(&b.ID, &b.BlockerUserID, &b.BlockedPartnerID, &b.Reason, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// IsArtistBlocked checks if a user has blocked a given partner.
func (r *BlockRepo) IsArtistBlocked(ctx context.Context, blockerUserID, blockedPartnerID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_artist_blocks WHERE blocker_user_id = $1::uuid AND blocked_partner_id = $2::uuid)`,
		blockerUserID, blockedPartnerID).Scan(&exists)
	return exists, err
}
