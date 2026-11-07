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
	var resolvedAt sql.NullTime
	err := row.Scan(&d.ID, &d.TrackID, &reporter, &d.Reason, &d.Description, &d.Status,
		&d.StrikeApplied, &resolver, &resolvedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.ReporterUserID = strPtr(reporter)
	d.ResolvedBy = strPtr(resolver)
	d.ResolvedAt = timePtr(resolvedAt)
	return d, nil
}

const disputeColumns = `id::text, track_id::text, reporter_user_id::text, reason, description,
	status, strike_applied, resolved_by::text, resolved_at, created_at, updated_at`

func (r *DisputeRepo) Create(ctx context.Context, trackID string, reporterID *string, reason, desc string) (*domain.CopyrightDispute, error) {
	d, err := scanDispute(r.db.QueryRowContext(ctx, `
		INSERT INTO copyright_disputes (track_id, reporter_user_id, reason, description)
		VALUES ($1::uuid, $2::uuid, $3, $4)
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
func (r *DisputeRepo) AppendCounter(ctx context.Context, id, note string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE copyright_disputes
		SET description = description || $2, status = 'UNDER_REVIEW'
		WHERE id = $1::uuid`, id, "\n[COUNTER] "+note)
	return err
}

// Resolve records an admin resolution decision.
func (r *DisputeRepo) Resolve(ctx context.Context, id, status, resolverID string, strikeApplied bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE copyright_disputes
		SET status = $2, strike_applied = $3, resolved_by = $4::uuid, resolved_at = NOW()
		WHERE id = $1::uuid`, id, status, strikeApplied, resolverID)
	return err
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
