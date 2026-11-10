package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"avibe-platform/backend/internal/domain"
)

// AuditRepo persists the append-only admin action trail.
type AuditRepo struct{ db *sql.DB }

// Record appends one audit event. metadataJSON must be a JSON object string
// (callers pass "{}" when there is nothing to record).
func (r *AuditRepo) Record(ctx context.Context, actorUserID *string, action, entityType, entityID, metadataJSON, ip string) error {
	if metadataJSON == "" {
		metadataJSON = "{}"
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_events (actor_user_id, action, entity_type, entity_id, metadata, ip)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6)`,
		nullStr(actorUserID), action, entityType, entityID, metadataJSON, ip)
	return err
}

// List returns audit events newest-first with optional action/entity filters.
func (r *AuditRepo) List(ctx context.Context, action, entityType string, limit, offset int) ([]*domain.AuditEvent, error) {
	limit = clampLimit(limit, 50, 200)
	if offset < 0 {
		offset = 0
	}
	q := `SELECT id, actor_user_id, action, entity_type, entity_id, metadata::text, ip, created_at
	      FROM audit_events WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(action) != "" {
		q += ` AND action = $` + strconv.Itoa(len(args)+1)
		args = append(args, strings.TrimSpace(action))
	}
	if strings.TrimSpace(entityType) != "" {
		q += ` AND entity_type = $` + strconv.Itoa(len(args)+1)
		args = append(args, strings.TrimSpace(entityType))
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AuditEvent{}
	for rows.Next() {
		var e domain.AuditEvent
		var actor sql.NullString
		if err := rows.Scan(&e.ID, &actor, &e.Action, &e.EntityType, &e.EntityID, &e.Metadata, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ActorUserID = strPtr(actor)
		out = append(out, &e)
	}
	return out, rows.Err()
}
