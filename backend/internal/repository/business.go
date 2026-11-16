package repository

import (
	"context"
	"database/sql"
	"fmt"

	"avibe-platform/backend/internal/domain"
)

// BusinessRepo accesses B2B inquiries and the generation waitlist.
type BusinessRepo struct{ db *sql.DB }

const inquiryColumns = `id::text, company, country, store_count, use_case,
	contact_name, contact_email, contact_phone, message, status, created_at, updated_at`

func scanInquiry(row interface{ Scan(...any) error }) (*domain.BusinessInquiry, error) {
	q := &domain.BusinessInquiry{}
	err := row.Scan(&q.ID, &q.Company, &q.Country, &q.StoreCount, &q.UseCase,
		&q.ContactName, &q.ContactEmail, &q.ContactPhone, &q.Message, &q.Status,
		&q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// CreateInquiry stores one inquiry with status NEW.
func (r *BusinessRepo) CreateInquiry(ctx context.Context, q *domain.BusinessInquiry) (*domain.BusinessInquiry, error) {
	return scanInquiry(r.db.QueryRowContext(ctx, `
		INSERT INTO business_inquiries (company, country, store_count, use_case,
			contact_name, contact_email, contact_phone, message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+inquiryColumns,
		q.Company, q.Country, q.StoreCount, q.UseCase,
		q.ContactName, q.ContactEmail, q.ContactPhone, q.Message))
}

// ListInquiries returns newest-first inquiries, optionally filtered by status.
func (r *BusinessRepo) ListInquiries(ctx context.Context, status string, limit int) ([]*domain.BusinessInquiry, error) {
	query := `SELECT ` + inquiryColumns + ` FROM business_inquiries`
	args := []any{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d`, len(args)+1)
	args = append(args, clampLimit(limit, 20, 100))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.BusinessInquiry{}
	for rows.Next() {
		q, err := scanInquiry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// SetInquiryStatus moves one inquiry; unknown ids yield ErrNotFound.
func (r *BusinessRepo) SetInquiryStatus(ctx context.Context, id, status string) (*domain.BusinessInquiry, error) {
	q, err := scanInquiry(r.db.QueryRowContext(ctx, `
		UPDATE business_inquiries SET status = $2, updated_at = NOW()
		WHERE id = $1::uuid RETURNING `+inquiryColumns, id, status))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return q, err
}

const businessProfileColumns = `user_id::text, company_name, biz_no, country, industry,
	status, verified_by::text, verified_at, created_at, updated_at`

func scanBusinessProfile(row interface{ Scan(...any) error }) (*domain.BusinessProfile, error) {
	p := &domain.BusinessProfile{}
	var bizNo, verifiedBy sql.NullString
	var verifiedAt sql.NullTime
	err := row.Scan(&p.UserID, &p.CompanyName, &bizNo, &p.Country, &p.Industry,
		&p.Status, &verifiedBy, &verifiedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.BizNo = strPtr(bizNo)
	p.VerifiedBy = strPtr(verifiedBy)
	p.VerifiedAt = timePtr(verifiedAt)
	return p, nil
}

// CreateBusinessProfile stores a BUSINESS account's company profile (R10).
func (r *BusinessRepo) CreateBusinessProfile(ctx context.Context, userID, company, bizNo, country, industry, status string) (*domain.BusinessProfile, error) {
	var biz any
	if bizNo != "" {
		biz = bizNo
	}
	return scanBusinessProfile(r.db.QueryRowContext(ctx, `
		INSERT INTO business_profiles (user_id, company_name, biz_no, country, industry, status)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING `+businessProfileColumns,
		userID, company, biz, country, industry, status))
}

// GetBusinessProfileByUser returns one account's business profile.
func (r *BusinessRepo) GetBusinessProfileByUser(ctx context.Context, userID string) (*domain.BusinessProfile, error) {
	p, err := scanBusinessProfile(r.db.QueryRowContext(ctx,
		`SELECT `+businessProfileColumns+` FROM business_profiles WHERE user_id = $1::uuid`, userID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// ListBusinessProfilesByStatus returns profiles filtered by status (admin).
func (r *BusinessRepo) ListBusinessProfilesByStatus(ctx context.Context, status string, limit int) ([]*domain.BusinessProfile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+businessProfileColumns+`
		FROM business_profiles WHERE status = $1 ORDER BY created_at DESC LIMIT $2`,
		status, clampLimit(limit, 20, 100))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.BusinessProfile{}
	for rows.Next() {
		p, err := scanBusinessProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ReviewBusinessProfile approves or suspends a profile (admin manual review).
func (r *BusinessRepo) ReviewBusinessProfile(ctx context.Context, userID, status, verifierID string) (*domain.BusinessProfile, error) {
	p, err := scanBusinessProfile(r.db.QueryRowContext(ctx, `
		UPDATE business_profiles
		SET status = $2, verified_by = $3::uuid, verified_at = NOW(), updated_at = NOW()
		WHERE user_id = $1::uuid RETURNING `+businessProfileColumns, userID, status, verifierID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// JoinWaitlist registers an email; re-registration returns existed=true.
func (r *BusinessRepo) JoinWaitlist(ctx context.Context, email, interest, locale string) (*domain.WaitlistEntry, bool, error) {
	w := &domain.WaitlistEntry{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO create_waitlist (email, interest, locale) VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING
		RETURNING id::text, email, interest, locale, created_at`,
		email, interest, locale).Scan(&w.ID, &w.Email, &w.Interest, &w.Locale, &w.CreatedAt)
	if err == nil {
		return w, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	err = r.db.QueryRowContext(ctx, `
		SELECT id::text, email, interest, locale, created_at FROM create_waitlist WHERE email = $1`,
		email).Scan(&w.ID, &w.Email, &w.Interest, &w.Locale, &w.CreatedAt)
	if err != nil {
		return nil, false, err
	}
	return w, true, nil
}
