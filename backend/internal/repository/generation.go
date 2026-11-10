package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// GenerationJobRepo accesses generation_jobs (WI-03).
type GenerationJobRepo struct{ db DBTX }

const generationJobColumns = `id::text, partner_id::text, prompt, lyrics, model,
	license_ref, params_json, status, output_object_key, duration_ms, error, attempts,
	created_at, updated_at`

func scanGenerationJob(row interface{ Scan(...any) error }) (*domain.GenerationJob, error) {
	g := &domain.GenerationJob{}
	var lyrics, outputKey, errmsg sql.NullString
	var duration sql.NullInt64
	err := row.Scan(&g.ID, &g.PartnerID, &g.Prompt, &lyrics, &g.Model,
		&g.LicenseRef, &g.ParamsJSON, &g.Status, &outputKey, &duration, &errmsg, &g.Attempts,
		&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.Lyrics = strPtr(lyrics)
	g.OutputObjectKey = strPtr(outputKey)
	g.DurationMs = intPtr(duration)
	g.Error = strPtr(errmsg)
	return g, nil
}

// Create inserts a QUEUED generation job.
func (r *GenerationJobRepo) Create(ctx context.Context, partnerID, prompt string, lyrics *string, model, paramsJSON string) (*domain.GenerationJob, error) {
	return r.CreateFull(ctx, partnerID, prompt, lyrics, model, "", paramsJSON)
}

// CreateFull inserts a QUEUED generation job with a license reference.
func (r *GenerationJobRepo) CreateFull(ctx context.Context, partnerID, prompt string, lyrics *string, model, licenseRef, paramsJSON string) (*domain.GenerationJob, error) {
	g, err := scanGenerationJob(r.db.QueryRowContext(ctx, `
		INSERT INTO generation_jobs (partner_id, prompt, lyrics, model, license_ref, params_json)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING `+generationJobColumns, partnerID, prompt, nullStr(lyrics), model, licenseRef, paramsJSON))
	return g, err
}

// CountTodayByPartner counts jobs filed since day (quota input).
func (r *GenerationJobRepo) CountTodayByPartner(ctx context.Context, partnerID string, day time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM generation_jobs WHERE partner_id = $1::uuid AND created_at >= $2`,
		partnerID, day).Scan(&n)
	return n, err
}

// RecordViolation logs a banned-prompt strike.
func (r *GenerationJobRepo) RecordViolation(ctx context.Context, partnerID, keyword string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO generation_violations (partner_id, matched_keyword) VALUES ($1::uuid, $2)`,
		partnerID, keyword)
	return err
}

// CountRecentViolations counts strikes since a time.
func (r *GenerationJobRepo) CountRecentViolations(ctx context.Context, partnerID string, since time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM generation_violations WHERE partner_id = $1::uuid AND created_at >= $2`,
		partnerID, since).Scan(&n)
	return n, err
}

// GetByID returns a generation job by ID.
func (r *GenerationJobRepo) GetByID(ctx context.Context, id string) (*domain.GenerationJob, error) {
	g, err := scanGenerationJob(r.db.QueryRowContext(ctx,
		`SELECT `+generationJobColumns+` FROM generation_jobs WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return g, err
}

// ListByPartner returns a partner's jobs, newest first.
func (r *GenerationJobRepo) ListByPartner(ctx context.Context, partnerID string, limit int) ([]*domain.GenerationJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+generationJobColumns+` FROM generation_jobs
		WHERE partner_id = $1::uuid
		ORDER BY created_at DESC LIMIT $2`, partnerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.GenerationJob{}
	for rows.Next() {
		g, err := scanGenerationJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// MarkRunning transitions a job to RUNNING.
func (r *GenerationJobRepo) MarkRunning(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE generation_jobs SET status = 'RUNNING', updated_at = NOW() WHERE id = $1::uuid`, id)
	return err
}

// MarkSucceeded transitions a job to SUCCEEDED with the staged output key.
func (r *GenerationJobRepo) MarkSucceeded(ctx context.Context, id, outputKey string, durationMs int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs SET status = 'SUCCEEDED', output_object_key = $2,
			duration_ms = $3, updated_at = NOW() WHERE id = $1::uuid`, id, outputKey, durationMs)
	return err
}

// MarkFailed transitions a job to FAILED and bumps attempts.
func (r *GenerationJobRepo) MarkFailed(ctx context.Context, id, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs SET status = 'FAILED', error = $2,
			attempts = attempts + 1, updated_at = NOW() WHERE id = $1::uuid`, id, errMsg)
	return err
}
