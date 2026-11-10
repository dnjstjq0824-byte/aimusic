package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
)

// UserRepo accesses the users table.
type UserRepo struct{ db *sql.DB }

const userColumns = `id::text, email, password_hash, role, display_name, is_active,
	birth_year, age_confirmed_at, token_version, totp_secret, totp_enabled,
	totp_enrolled_at, COALESCE(array_length(totp_recovery_hashes, 1), 0),
	created_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (*domain.User, error) {
	u := &domain.User{}
	var birth sql.NullInt64
	var ageConfirmed, totpEnrolled sql.NullTime
	var totpSecret sql.NullString
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName,
		&u.IsActive, &birth, &ageConfirmed, &u.TokenVersion,
		&totpSecret, &u.TOTPEnabled, &totpEnrolled, &u.TOTPRecoveryLeft,
		&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.BirthYear = intPtr(birth)
	u.AgeConfirmedAt = timePtr(ageConfirmed)
	u.TOTPEnrolledAt = timePtr(totpEnrolled)
	if totpSecret.Valid {
		u.TOTPSecret = totpSecret.String
	}
	return u, nil
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, role, displayName string) (*domain.User, error) {
	return r.CreateWithProfile(ctx, email, passwordHash, role, displayName, nil, false)
}

// CreateWithProfile creates a user with 14+ age-gate evidence (birth year is
// optional; ageConfirmed stamps age_confirmed_at).
func (r *UserRepo) CreateWithProfile(ctx context.Context, email, passwordHash, role, displayName string, birthYear *int, ageConfirmed bool) (*domain.User, error) {
	var birth any
	if birthYear != nil {
		birth = *birthYear
	}
	var confirmed any
	if ageConfirmed {
		confirmed = time.Now()
	}
	u, err := scanUser(r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, display_name, birth_year, age_confirmed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+userColumns,
		email, passwordHash, role, displayName, birth, confirmed))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
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

// SetRole changes an account's role (operator action; see cmd/adminctl).
func (r *UserRepo) SetRole(ctx context.Context, id, role string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET role = $2 WHERE id = $1::uuid`, id, role)
	return err
}

// BumpTokenVersion invalidates every outstanding JWT for the account.
func (r *UserRepo) BumpTokenVersion(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET token_version = token_version + 1 WHERE id = $1::uuid`, id)
	return err
}

// PurgePII anonymizes identity columns on withdrawal (privacy policy: purge
// personal data except tax/legal retention rows, which stay pseudonymous).
// The email is freed for re-registration via a deterministic deleted alias.
func (r *UserRepo) PurgePII(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET
			email = 'deleted+' || id::text || '@deleted.avibe.local',
			display_name = '탈퇴한 사용자',
			password_hash = '!',
			birth_year = NULL,
			totp_secret = NULL, totp_enabled = FALSE, totp_enrolled_at = NULL,
			totp_recovery_hashes = '{}'
		WHERE id = $1::uuid`, id)
	return err
}

// ConfirmAge stamps the 14+ self-declaration (post-OAuth age gate).
func (r *UserRepo) ConfirmAge(ctx context.Context, id string, birthYear *int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET birth_year = COALESCE($2, birth_year), age_confirmed_at = NOW() WHERE id = $1::uuid`,
		id, nullInt(birthYear))
	return err
}

// StageTOTPSecret stores a pending enrollment secret (not yet enabled).
func (r *UserRepo) StageTOTPSecret(ctx context.Context, id, secret string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET totp_secret = $2, totp_enabled = FALSE, totp_enrolled_at = NULL WHERE id = $1::uuid`,
		id, secret)
	return err
}

// EnableTOTP activates the staged secret and stores recovery-code hashes.
func (r *UserRepo) EnableTOTP(ctx context.Context, id, secret string, recoveryHashes []string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET totp_secret = $2, totp_enabled = TRUE, totp_enrolled_at = NOW(),
			totp_recovery_hashes = string_to_array($3, chr(31))
		WHERE id = $1::uuid`, id, secret, joinKeywords(recoveryHashes))
	return err
}

// DisableTOTP removes the second factor entirely.
func (r *UserRepo) DisableTOTP(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET totp_secret = NULL, totp_enabled = FALSE, totp_enrolled_at = NULL,
			totp_recovery_hashes = '{}'
		WHERE id = $1::uuid`, id)
	return err
}

// TOTPRecoveryHashes returns the stored bcrypt hashes of unused recovery codes.
func (r *UserRepo) TOTPRecoveryHashes(ctx context.Context, id string) ([]string, error) {
	var joined sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(array_to_string(totp_recovery_hashes, chr(31)), '') FROM users WHERE id = $1::uuid`,
		id).Scan(&joined)
	if err != nil {
		return nil, err
	}
	if !joined.Valid || joined.String == "" {
		return []string{}, nil
	}
	return splitKeywords(joined.String), nil
}

// ConsumeTOTPRecoveryHash removes one used recovery hash.
func (r *UserRepo) ConsumeTOTPRecoveryHash(ctx context.Context, id, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET totp_recovery_hashes = array_remove(totp_recovery_hashes, $2)
		WHERE id = $1::uuid`, id, hash)
	return err
}

// CountAll returns the total user count.
func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// Search returns users filtered by email/display substring and role,
// newest first (admin user management).
func (r *UserRepo) Search(ctx context.Context, q, role string, limit, offset int) ([]*domain.User, error) {
	limit = clampLimit(limit, 20, 100)
	if offset < 0 {
		offset = 0
	}
	query := `SELECT ` + userColumns + ` FROM users WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(q) != "" {
		query += ` AND (email ILIKE $` + strconv.Itoa(len(args)+1) + ` OR display_name ILIKE $` + strconv.Itoa(len(args)+1) + `)`
		args = append(args, "%"+strings.TrimSpace(q)+"%")
	}
	if strings.TrimSpace(role) != "" {
		query += ` AND role = $` + strconv.Itoa(len(args)+1)
		args = append(args, strings.TrimSpace(role))
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountByRole returns the user count for one role.
func (r *UserRepo) CountByRole(ctx context.Context, role string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, role).Scan(&n)
	return n, err
}

// ListIDsByRole returns active user IDs for one role (SLA alert fan-out).
func (r *UserRepo) ListIDsByRole(ctx context.Context, role string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id::text FROM users WHERE role = $1 AND is_active ORDER BY created_at ASC`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
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

// PurgePIIByUser clears partner contact/payout data on withdrawal. Company
// name and settlement rows stay for tax retention.
func (r *PartnerRepo) PurgePIIByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE partner_profiles SET contact_email = 'deleted+' || user_id::text || '@deleted.avibe.local',
			business_number = NULL, settlement_bank = NULL, settlement_account_encrypted = NULL
		WHERE user_id = $1::uuid`, userID)
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
	var terms, snapshot sql.NullString
	var maxGen, termsChecked sql.NullTime
	err := row.Scan(&p.ID, &p.Provider, &p.AllowsStandaloneStreaming,
		&allowedJoined, &blockedTiersJoined, &syncOnlyJoined,
		&p.MinBitrateKbps, &joined, &p.RejectionMessageKo, &terms,
		&maxGen, &snapshot, &termsChecked, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.AllowedPlanTiers = splitKeywords(allowedJoined)
	p.BlockedPlanTiers = splitKeywords(blockedTiersJoined)
	p.SyncOnlyPlanTiers = splitKeywords(syncOnlyJoined)
	p.BlockedMetadataKeywords = splitKeywords(joined)
	p.CommercialTermsURL = strPtr(terms)
	p.MaxGeneratedAt = timePtr(maxGen)
	p.TermsSnapshotURL = strPtr(snapshot)
	p.TermsCheckedAt = timePtr(termsChecked)
	return p, nil
}

const policyColumns = `id::text, provider, allows_standalone_streaming,
	COALESCE(array_to_string(allowed_plan_tiers, chr(31)), ''),
	COALESCE(array_to_string(blocked_plan_tiers, chr(31)), ''),
	COALESCE(array_to_string(sync_only_plan_tiers, chr(31)), ''),
	min_bitrate_kbps,
	COALESCE(array_to_string(blocked_metadata_keywords, chr(31)), ''), rejection_message_ko,
	commercial_terms_url, max_generated_at, terms_snapshot_url, terms_checked_at, created_at, updated_at`

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
// Nil maxGeneratedAt/termsSnapshotURL preserve the current values; every save
// stamps terms_checked_at to record the admin review time.
func (r *PolicyRepo) Update(ctx context.Context, providerOrID string, allows bool, minKbps int, blocked []string, rejectionKo string, termsURL *string, allowedTiers, blockedTiers, syncOnlyTiers []string, maxGeneratedAt *time.Time, termsSnapshotURL *string) (*domain.AIProviderPolicy, error) {
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
	if maxGeneratedAt == nil {
		maxGeneratedAt = cur.MaxGeneratedAt
	}
	if termsSnapshotURL == nil {
		termsSnapshotURL = cur.TermsSnapshotURL
	}
	p, err := scanPolicy(r.db.QueryRowContext(ctx, `
		UPDATE ai_provider_policies
		SET allows_standalone_streaming = $2, min_bitrate_kbps = $3,
			blocked_metadata_keywords = string_to_array($4, chr(31)),
			rejection_message_ko = $5, commercial_terms_url = $6,
			allowed_plan_tiers = string_to_array($7, chr(31)),
			blocked_plan_tiers = string_to_array($8, chr(31)),
			sync_only_plan_tiers = string_to_array($9, chr(31)),
			max_generated_at = $10, terms_snapshot_url = $11, terms_checked_at = NOW()
		WHERE provider = $1
		RETURNING `+policyColumns,
		cur.Provider, allows, minKbps, joinKeywords(blocked), rejectionKo, nullStr(termsURL),
		joinKeywords(allowedTiers), joinKeywords(blockedTiers), joinKeywords(syncOnlyTiers),
		nullTime(maxGeneratedAt), nullStr(termsSnapshotURL)))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}
