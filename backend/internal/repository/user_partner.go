package repository

import (
	"context"
	"database/sql"
	"time"

	"ai-music-platform/backend/internal/domain"
)

// UserRepo accesses the users table.
type UserRepo struct{ db *sql.DB }

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, role, displayName string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, email, password_hash, role, display_name, is_active, created_at, updated_at`,
		email, passwordHash, role, displayName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text, email, password_hash, role, display_name, is_active, created_at, updated_at
		FROM users WHERE id = $1::uuid`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text, email, password_hash, role, display_name, is_active, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SetActive enables or disables an account.
func (r *UserRepo) SetActive(ctx context.Context, id string, active bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_active = $2 WHERE id = $1::uuid`, id, active)
	return err
}

// UpdatePasswordHash replaces an account's password hash.
func (r *UserRepo) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1::uuid`, id, hash)
	return err
}

// CountAll returns the total user count.
func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// CountByRole returns the user count for one role.
func (r *UserRepo) CountByRole(ctx context.Context, role string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, role).Scan(&n)
	return n, err
}

// PartnerRepo accesses partner_profiles.
type PartnerRepo struct{ db *sql.DB }

func scanPartner(row interface{ Scan(...any) error }) (*domain.PartnerProfile, error) {
	p := &domain.PartnerProfile{}
	var biz, bank, enc sql.NullString
	var approved sql.NullTime
	err := row.Scan(&p.ID, &p.UserID, &p.CompanyName, &biz, &bank, &enc,
		&p.ContactEmail, &p.Status, &approved, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.BusinessNumber = strPtr(biz)
	p.SettlementBank = strPtr(bank)
	p.SettlementAccountEncrypted = strPtr(enc)
	p.ApprovedAt = timePtr(approved)
	return p, nil
}

const partnerColumns = `id::text, user_id::text, company_name, business_number, settlement_bank,
	settlement_account_encrypted, contact_email, status, approved_at, created_at, updated_at`

func (r *PartnerRepo) Create(ctx context.Context, userID, companyName, contactEmail string) (*domain.PartnerProfile, error) {
	p, err := scanPartner(r.db.QueryRowContext(ctx, `
		INSERT INTO partner_profiles (user_id, company_name, contact_email)
		VALUES ($1::uuid, $2, $3)
		RETURNING `+partnerColumns, userID, companyName, contactEmail))
	return p, err
}

func (r *PartnerRepo) GetByID(ctx context.Context, id string) (*domain.PartnerProfile, error) {
	p, err := scanPartner(r.db.QueryRowContext(ctx,
		`SELECT `+partnerColumns+` FROM partner_profiles WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

func (r *PartnerRepo) GetByUserID(ctx context.Context, userID string) (*domain.PartnerProfile, error) {
	p, err := scanPartner(r.db.QueryRowContext(ctx,
		`SELECT `+partnerColumns+` FROM partner_profiles WHERE user_id = $1::uuid`, userID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// CountByStatus returns the partner count for one status.
func (r *PartnerRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM partner_profiles WHERE status = $1`, status).Scan(&n)
	return n, err
}

// ListByStatus returns partner profiles filtered by status.
func (r *PartnerRepo) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*domain.PartnerProfile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+partnerColumns+`
		FROM partner_profiles WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		status, clampLimit(limit, 20, 100), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.PartnerProfile
	for rows.Next() {
		p, err := scanPartner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateStatus transitions a partner; approved=true stamps approved_at.
func (r *PartnerRepo) UpdateStatus(ctx context.Context, id, status string, approved bool) error {
	if approved {
		_, err := r.db.ExecContext(ctx,
			`UPDATE partner_profiles SET status = $2, approved_at = NOW() WHERE id = $1::uuid`, id, status)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE partner_profiles SET status = $2 WHERE id = $1::uuid`, id, status)
	return err
}

// UpdateSettlement stores bank and encrypted account reference.
func (r *PartnerRepo) UpdateSettlement(ctx context.Context, id, bank, encAccount string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE partner_profiles SET settlement_bank = $2, settlement_account_encrypted = $3 WHERE id = $1::uuid`,
		id, bank, encAccount)
	return err
}

// SubscriptionRepo accesses partner_ai_subscriptions.
type SubscriptionRepo struct{ db *sql.DB }

func scanSubscription(row interface{ Scan(...any) error }) (*domain.PartnerAISubscription, error) {
	s := &domain.PartnerAISubscription{}
	var proof, verifier sql.NullString
	var verifiedAt, expiresAt sql.NullTime
	err := row.Scan(&s.ID, &s.PartnerID, &s.Provider, &s.PlanName, &proof,
		&s.Status, &verifier, &verifiedAt, &expiresAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.ProofObjectKey = strPtr(proof)
	s.VerifiedBy = strPtr(verifier)
	s.VerifiedAt = timePtr(verifiedAt)
	s.ExpiresAt = timePtr(expiresAt)
	return s, nil
}

const subscriptionColumns = `id::text, partner_id::text, provider, plan_name, proof_object_key,
	status, verified_by::text, verified_at, expires_at, created_at, updated_at`

func (r *SubscriptionRepo) Create(ctx context.Context, partnerID, provider, planName string, proofKey *string) (*domain.PartnerAISubscription, error) {
	s, err := scanSubscription(r.db.QueryRowContext(ctx, `
		INSERT INTO partner_ai_subscriptions (partner_id, provider, plan_name, proof_object_key)
		VALUES ($1::uuid, $2, $3, $4)
		RETURNING `+subscriptionColumns, partnerID, provider, planName, nullStr(proofKey)))
	return s, err
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id string) (*domain.PartnerAISubscription, error) {
	s, err := scanSubscription(r.db.QueryRowContext(ctx,
		`SELECT `+subscriptionColumns+` FROM partner_ai_subscriptions WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return s, err
}

func (r *SubscriptionRepo) GetByPartnerProvider(ctx context.Context, partnerID, provider string) (*domain.PartnerAISubscription, error) {
	s, err := scanSubscription(r.db.QueryRowContext(ctx,
		`SELECT `+subscriptionColumns+` FROM partner_ai_subscriptions WHERE partner_id = $1::uuid AND provider = $2`,
		partnerID, provider))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return s, err
}

// ListByPartner returns all subscriptions of a partner.
func (r *SubscriptionRepo) ListByPartner(ctx context.Context, partnerID string) ([]*domain.PartnerAISubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+subscriptionColumns+`
		FROM partner_ai_subscriptions WHERE partner_id = $1::uuid ORDER BY created_at DESC`, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.PartnerAISubscription
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountByStatus returns the subscription count for one status.
func (r *SubscriptionRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM partner_ai_subscriptions WHERE status = $1`, status).Scan(&n)
	return n, err
}

// ListPending returns subscriptions awaiting admin verification.
func (r *SubscriptionRepo) ListPending(ctx context.Context, limit, offset int) ([]*domain.PartnerAISubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+subscriptionColumns+`
		FROM partner_ai_subscriptions WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT $1 OFFSET $2`,
		clampLimit(limit, 20, 100), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.PartnerAISubscription
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateVerification records an admin verification decision.
func (r *SubscriptionRepo) UpdateVerification(ctx context.Context, id, status, verifierID string, expiresAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE partner_ai_subscriptions
		SET status = $2, verified_by = $3::uuid, verified_at = NOW(), expires_at = $4
		WHERE id = $1::uuid`, id, status, verifierID, nullTime(expiresAt))
	return err
}

// PolicyRepo accesses ai_provider_policies.
type PolicyRepo struct{ db *sql.DB }

func scanPolicy(row interface{ Scan(...any) error }) (*domain.AIProviderPolicy, error) {
	p := &domain.AIProviderPolicy{}
	var allowedJoined, blockedTiersJoined, syncOnlyJoined, joined string
	var terms sql.NullString
	err := row.Scan(&p.ID, &p.Provider, &p.AllowsStandaloneStreaming,
		&allowedJoined, &blockedTiersJoined, &syncOnlyJoined,
		&p.MinBitrateKbps, &joined, &p.RejectionMessageKo, &terms, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.AllowedPlanTiers = splitKeywords(allowedJoined)
	p.BlockedPlanTiers = splitKeywords(blockedTiersJoined)
	p.SyncOnlyPlanTiers = splitKeywords(syncOnlyJoined)
	p.BlockedMetadataKeywords = splitKeywords(joined)
	p.CommercialTermsURL = strPtr(terms)
	return p, nil
}

const policyColumns = `id::text, provider, allows_standalone_streaming,
	COALESCE(array_to_string(allowed_plan_tiers, chr(31)), ''),
	COALESCE(array_to_string(blocked_plan_tiers, chr(31)), ''),
	COALESCE(array_to_string(sync_only_plan_tiers, chr(31)), ''),
	min_bitrate_kbps,
	COALESCE(array_to_string(blocked_metadata_keywords, chr(31)), ''), rejection_message_ko,
	commercial_terms_url, created_at, updated_at`

func (r *PolicyRepo) Get(ctx context.Context, provider string) (*domain.AIProviderPolicy, error) {
	p, err := scanPolicy(r.db.QueryRowContext(ctx,
		`SELECT `+policyColumns+` FROM ai_provider_policies WHERE provider = $1 OR id::text = $1`, provider))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// List returns the full provider policy matrix.
func (r *PolicyRepo) List(ctx context.Context) ([]*domain.AIProviderPolicy, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+policyColumns+` FROM ai_provider_policies ORDER BY provider ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AIProviderPolicy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Update rewrites a provider policy row (matching by provider code or policy ID).
func (r *PolicyRepo) Update(ctx context.Context, providerOrID string, allows bool, minKbps int, blocked []string, rejectionKo string, termsURL *string, allowedTiers, blockedTiers, syncOnlyTiers []string) (*domain.AIProviderPolicy, error) {
	cur, err := r.Get(ctx, providerOrID)
	if err != nil {
		return nil, err
	}
	if allowedTiers == nil {
		allowedTiers = cur.AllowedPlanTiers
	}
	if blockedTiers == nil {
		blockedTiers = cur.BlockedPlanTiers
	}
	if syncOnlyTiers == nil {
		syncOnlyTiers = cur.SyncOnlyPlanTiers
	}
	p, err := scanPolicy(r.db.QueryRowContext(ctx, `
		UPDATE ai_provider_policies
		SET allows_standalone_streaming = $2, min_bitrate_kbps = $3,
			blocked_metadata_keywords = string_to_array($4, chr(31)),
			rejection_message_ko = $5, commercial_terms_url = $6,
			allowed_plan_tiers = string_to_array($7, chr(31)),
			blocked_plan_tiers = string_to_array($8, chr(31)),
			sync_only_plan_tiers = string_to_array($9, chr(31))
		WHERE provider = $1
		RETURNING `+policyColumns,
		cur.Provider, allows, minKbps, joinKeywords(blocked), rejectionKo, nullStr(termsURL),
		joinKeywords(allowedTiers), joinKeywords(blockedTiers), joinKeywords(syncOnlyTiers)))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}
