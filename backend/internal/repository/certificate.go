package repository

import (
	"context"
	"database/sql"

	"avibe-platform/backend/internal/domain"
)

// CertificateRepo accesses track_certificates.
type CertificateRepo struct{ db DBTX }

const certColumns = `id::text, track_id::text, cert_number, issued_by::text, issued_at, created_at`

func scanCertificate(row interface{ Scan(...any) error }) (*domain.TrackCertificate, error) {
	c := &domain.TrackCertificate{}
	var issuedBy sql.NullString
	if err := row.Scan(&c.ID, &c.TrackID, &c.CertNumber, &issuedBy, &c.IssuedAt, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.IssuedBy = strPtr(issuedBy)
	return c, nil
}

// GetByTrack returns a track's certificate (ErrNotFound when none).
func (r *CertificateRepo) GetByTrack(ctx context.Context, trackID string) (*domain.TrackCertificate, error) {
	c, err := scanCertificate(r.db.QueryRowContext(ctx,
		`SELECT `+certColumns+` FROM track_certificates WHERE track_id = $1::uuid`, trackID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return c, err
}

// GetByNumber returns a certificate by its public number (ErrNotFound when unknown).
func (r *CertificateRepo) GetByNumber(ctx context.Context, number string) (*domain.TrackCertificate, error) {
	c, err := scanCertificate(r.db.QueryRowContext(ctx,
		`SELECT `+certColumns+` FROM track_certificates WHERE cert_number = $1`, number))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return c, err
}

// Create issues a certificate with a caller-generated number.
func (r *CertificateRepo) Create(ctx context.Context, trackID, number, actorID string) (*domain.TrackCertificate, error) {
	return scanCertificate(r.db.QueryRowContext(ctx, `
		INSERT INTO track_certificates (track_id, cert_number, issued_by)
		VALUES ($1::uuid, $2, $3::uuid)
		RETURNING `+certColumns, trackID, number, actorID))
}
