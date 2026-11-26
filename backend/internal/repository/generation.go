package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"avibe-platform/backend/internal/domain"
)

// GenerationJobRepo accesses generation_jobs (WI-03).
type GenerationJobRepo struct{ db DBTX }

const generationJobColumns = `id::text, partner_id::text, user_id::text, prompt, lyrics, model,
	model_version, seed, license_ref, params_json, safety_result, watermark_id,
	status, output_object_key, duration_ms, error, attempts,
	created_at, updated_at`

func scanGenerationJob(row interface{ Scan(...any) error }) (*domain.GenerationJob, error) {
	g := &domain.GenerationJob{}
	var userID, lyrics, outputKey, errmsg sql.NullString
	var seed, duration sql.NullInt64
	err := row.Scan(&g.ID, &g.PartnerID, &userID, &g.Prompt, &lyrics, &g.Model,
		&g.ModelVersion, &seed, &g.LicenseRef, &g.ParamsJSON, &g.SafetyResult, &g.WatermarkID,
		&g.Status, &outputKey, &duration, &errmsg, &g.Attempts,
		&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.UserID = strPtr(userID)
	g.Lyrics = strPtr(lyrics)
	if seed.Valid {
		g.Seed = &seed.Int64
	}
	g.OutputObjectKey = strPtr(outputKey)
	g.DurationMs = intPtr(duration)
	g.Error = strPtr(errmsg)
	return g, nil
}

// CreateFullCapped inserts a QUEUED generation job with a license reference
// and optional reproducibility pins (seed/modelVersion, WI-G0), enforcing
// the daily cap atomically. The partner row is locked in its own statement
// inside an explicit transaction: the lock wait lands before the counting
// snapshot, so concurrent submits serialize and exactly cap rows win. (A
// single-statement CTE cannot do this — the whole statement shares one
// snapshot, so concurrent writers all count stale and all insert.)
// sql.ErrNoRows means cap reached; a missing partner is ErrNotFound.
func (r *GenerationJobRepo) CreateFullCapped(ctx context.Context, partnerID, prompt string, lyrics *string, model, licenseRef, paramsJSON string, seed *int64, modelVersion string, day time.Time, cap int) (*domain.GenerationJob, error) {
	// GenJobs never runs inside WithTx, so the handle is always the pool in
	// practice; the fallback keeps transactional callers serial-correct.
	if sqldb, ok := r.db.(*sql.DB); ok {
		tx, err := sqldb.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		var pid string
		if err := tx.QueryRowContext(ctx,
			`SELECT id::text FROM partner_profiles WHERE id = $1::uuid FOR UPDATE`, partnerID).Scan(&pid); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		g, err := scanGenerationJob(tx.QueryRowContext(ctx, `
			INSERT INTO generation_jobs (partner_id, prompt, lyrics, model, license_ref, params_json, seed, model_version)
			SELECT $1::uuid, $2, $3, $4, $5, $6, $7, $8
			WHERE (SELECT COUNT(*) FROM generation_jobs
			       WHERE partner_id = $1::uuid AND created_at >= $9) < $10
			RETURNING `+generationJobColumns,
			partnerID, prompt, nullStr(lyrics), model, licenseRef, paramsJSON, nullInt64(seed), modelVersion, day, cap))
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return g, nil
	}
	return scanGenerationJob(r.db.QueryRowContext(ctx, `
		INSERT INTO generation_jobs (partner_id, prompt, lyrics, model, license_ref, params_json, seed, model_version)
		SELECT $1::uuid, $2, $3, $4, $5, $6, $7, $8
		WHERE (SELECT COUNT(*) FROM generation_jobs
		       WHERE partner_id = $1::uuid AND created_at >= $9) < $10
		RETURNING `+generationJobColumns,
		partnerID, prompt, nullStr(lyrics), model, licenseRef, paramsJSON, nullInt64(seed), modelVersion, day, cap))
}

// UpdateResult records the contract-v2 inference evidence (WI-G1): the exact
// model version + seed served, the weights hash, the safety verdict JSON,
// and the watermark ID the AVIBE gate branch requires.
func (r *GenerationJobRepo) UpdateResult(ctx context.Context, id, modelVersion string, seed *int64, weightsHash, safetyJSON, watermarkID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs SET model_version = $2, seed = $3, safety_result = $4,
			watermark_id = $5, params_json = CASE WHEN $6 = '' THEN params_json
			ELSE ((COALESCE(NULLIF(params_json, '{}'), '{}'))::jsonb || jsonb_build_object('weightsHash', $6))::text END,
			updated_at = NOW() WHERE id = $1::uuid`,
		id, modelVersion, nullInt64(seed), safetyJSON, watermarkID, weightsHash)
	return err
}

// FindSucceededByOutputKey returns the SUCCEEDED job that staged a key, for
// the AVIBE finalize branch (WI-G0). Nil job (no error) means no provenance.
func (r *GenerationJobRepo) FindSucceededByOutputKey(ctx context.Context, partnerID, outputKey string) (*domain.GenerationJob, error) {
	g, err := scanGenerationJob(r.db.QueryRowContext(ctx,
		`SELECT `+generationJobColumns+` FROM generation_jobs
		WHERE partner_id = $1::uuid AND output_object_key = $2 AND status = 'SUCCEEDED'`, partnerID, outputKey))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return g, err
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

// ListStuckRunning returns RUNNING jobs untouched since the cutoff (worker
// died mid-job and the Redis payload is gone).
func (r *GenerationJobRepo) ListStuckRunning(ctx context.Context, cutoff time.Time, limit int) ([]*domain.GenerationJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+generationJobColumns+` FROM generation_jobs
		WHERE status = 'RUNNING' AND updated_at < $1
		ORDER BY updated_at ASC LIMIT $2`, cutoff, limit)
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

// RequeueStuck resets one RUNNING job to QUEUED (attempts+1). It reports
// whether the row was actually reset (a live worker may have finished it).
func (r *GenerationJobRepo) RequeueStuck(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs SET status = 'QUEUED', attempts = attempts + 1,
			error = NULL, updated_at = NOW()
		WHERE id = $1::uuid AND status = 'RUNNING'`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}
