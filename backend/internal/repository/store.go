package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// StoreRepo accesses store_businesses, store_devices, store_schedules, and
// store_play_logs (WI-05 #19/#20/#21).
type StoreRepo struct{ db DBTX }

const storeBusinessColumns = `id::text, partner_id::text, business_number, store_name,
	license_count, status, verified_by::text, verified_at, created_at, updated_at`

func scanStoreBusiness(row interface{ Scan(...any) error }) (*domain.StoreBusiness, error) {
	b := &domain.StoreBusiness{}
	var verifiedBy sql.NullString
	var verifiedAt sql.NullTime
	err := row.Scan(&b.ID, &b.PartnerID, &b.BusinessNumber, &b.StoreName,
		&b.LicenseCount, &b.Status, &verifiedBy, &verifiedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	b.VerifiedBy = strPtr(verifiedBy)
	b.VerifiedAt = timePtr(verifiedAt)
	return b, nil
}

// CreateBusiness registers a store business as PENDING verification.
func (r *StoreRepo) CreateBusiness(ctx context.Context, partnerID, businessNumber, storeName string, licenseCount int) (*domain.StoreBusiness, error) {
	b, err := scanStoreBusiness(r.db.QueryRowContext(ctx, `
		INSERT INTO store_businesses (partner_id, business_number, store_name, license_count)
		VALUES ($1::uuid, $2, $3, $4)
		RETURNING `+storeBusinessColumns, partnerID, businessNumber, storeName, licenseCount))
	return b, err
}

// GetBusinessByID returns one store business.
func (r *StoreRepo) GetBusinessByID(ctx context.Context, id string) (*domain.StoreBusiness, error) {
	b, err := scanStoreBusiness(r.db.QueryRowContext(ctx,
		`SELECT `+storeBusinessColumns+` FROM store_businesses WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return b, err
}

// ListBusinessesByPartner returns a partner's store businesses.
func (r *StoreRepo) ListBusinessesByPartner(ctx context.Context, partnerID string) ([]*domain.StoreBusiness, error) {
	return r.listBusinesses(ctx,
		`SELECT `+storeBusinessColumns+` FROM store_businesses WHERE partner_id = $1::uuid ORDER BY created_at DESC`, partnerID)
}

// ListBusinessesByStatus returns store businesses filtered by status.
func (r *StoreRepo) ListBusinessesByStatus(ctx context.Context, status string, limit, offset int) ([]*domain.StoreBusiness, error) {
	return r.listBusinesses(ctx,
		`SELECT `+storeBusinessColumns+` FROM store_businesses WHERE status = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
		status, clampLimit(limit, 20, 100), offset)
}

func (r *StoreRepo) listBusinesses(ctx context.Context, query string, args ...any) ([]*domain.StoreBusiness, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.StoreBusiness{}
	for rows.Next() {
		b, err := scanStoreBusiness(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// VerifyBusiness approves or suspends a store business.
func (r *StoreRepo) VerifyBusiness(ctx context.Context, id, status, verifierID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE store_businesses SET status = $2, verified_by = $3::uuid, verified_at = NOW(), updated_at = NOW()
		WHERE id = $1::uuid`, id, status, verifierID)
	return err
}

const storeDeviceColumns = `id::text, business_id::text, device_name, pairing_code,
	pairing_expires_at, device_token_hash, status, volume, last_seen_at, created_at, updated_at`

func scanStoreDevice(row interface{ Scan(...any) error }) (*domain.StoreDevice, error) {
	d := &domain.StoreDevice{}
	var pairing, hash sql.NullString
	var pairingExp, seen sql.NullTime
	err := row.Scan(&d.ID, &d.BusinessID, &d.DeviceName, &pairing,
		&pairingExp, &hash, &d.Status, &d.Volume, &seen, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.PairingCode = strPtr(pairing)
	d.PairingExpiresAt = timePtr(pairingExp)
	d.DeviceTokenHash = strPtr(hash)
	d.LastSeenAt = timePtr(seen)
	return d, nil
}

// CreateDevice registers a PENDING player device with a pairing code.
func (r *StoreRepo) CreateDevice(ctx context.Context, businessID, deviceName, pairingCode string, pairingExpiry time.Time) (*domain.StoreDevice, error) {
	d, err := scanStoreDevice(r.db.QueryRowContext(ctx, `
		INSERT INTO store_devices (business_id, device_name, pairing_code, pairing_expires_at)
		VALUES ($1::uuid, $2, $3, $4)
		RETURNING `+storeDeviceColumns, businessID, deviceName, pairingCode, pairingExpiry))
	return d, err
}

// GetDeviceByID returns one player device.
func (r *StoreRepo) GetDeviceByID(ctx context.Context, id string) (*domain.StoreDevice, error) {
	d, err := scanStoreDevice(r.db.QueryRowContext(ctx,
		`SELECT `+storeDeviceColumns+` FROM store_devices WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return d, err
}

// GetDeviceByPairingCode returns the PENDING device holding a live code.
func (r *StoreRepo) GetDeviceByPairingCode(ctx context.Context, code string) (*domain.StoreDevice, error) {
	d, err := scanStoreDevice(r.db.QueryRowContext(ctx, `
		SELECT `+storeDeviceColumns+` FROM store_devices
		WHERE status = 'PENDING' AND pairing_code = $1 AND pairing_expires_at > NOW()`, code))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return d, err
}

// GetDeviceByTokenHash returns the ACTIVE device for a token hash.
func (r *StoreRepo) GetDeviceByTokenHash(ctx context.Context, hash string) (*domain.StoreDevice, error) {
	d, err := scanStoreDevice(r.db.QueryRowContext(ctx, `
		SELECT `+storeDeviceColumns+` FROM store_devices
		WHERE status = 'ACTIVE' AND device_token_hash = $1`, hash))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return d, err
}

// ListDevicesByBusiness returns a business's player devices.
func (r *StoreRepo) ListDevicesByBusiness(ctx context.Context, businessID string) ([]*domain.StoreDevice, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+storeDeviceColumns+` FROM store_devices
		WHERE business_id = $1::uuid ORDER BY created_at DESC`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.StoreDevice{}
	for rows.Next() {
		d, err := scanStoreDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ActivateDevice claims a pairing code: clears it, stores the token hash.
func (r *StoreRepo) ActivateDevice(ctx context.Context, id, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE store_devices SET status = 'ACTIVE', device_token_hash = $2,
			pairing_code = NULL, pairing_expires_at = NULL, updated_at = NOW()
		WHERE id = $1::uuid AND status = 'PENDING'`, id, tokenHash)
	return err
}

// UpdateDevice sets name/volume on a device.
func (r *StoreRepo) UpdateDevice(ctx context.Context, id, deviceName string, volume int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE store_devices SET device_name = $2, volume = $3, updated_at = NOW()
		WHERE id = $1::uuid`, id, deviceName, volume)
	return err
}

// RevokeDevice deactivates a player device.
func (r *StoreRepo) RevokeDevice(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE store_devices SET status = 'REVOKED', device_token_hash = NULL, updated_at = NOW()
		WHERE id = $1::uuid`, id)
	return err
}

// TouchDeviceSeen stamps device activity.
func (r *StoreRepo) TouchDeviceSeen(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE store_devices SET last_seen_at = NOW() WHERE id = $1::uuid`, id)
	return err
}

const storeScheduleColumns = `id::text, business_id::text, weekday, open_time, close_time,
	playlist_id::text, is_closed, created_at, updated_at`

func scanStoreSchedule(row interface{ Scan(...any) error }) (*domain.StoreSchedule, error) {
	s := &domain.StoreSchedule{}
	var openT, closeT time.Time
	var playlist sql.NullString
	err := row.Scan(&s.ID, &s.BusinessID, &s.Weekday, &openT, &closeT,
		&playlist, &s.IsClosed, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.OpenTime = openT.Format("15:04")
	s.CloseTime = closeT.Format("15:04")
	s.PlaylistID = strPtr(playlist)
	return s, nil
}

// UpsertSchedule sets one weekday row (open/close as "15:04").
func (r *StoreRepo) UpsertSchedule(ctx context.Context, businessID string, weekday int, open, closeTime string, playlistID *string, isClosed bool) (*domain.StoreSchedule, error) {
	s, err := scanStoreSchedule(r.db.QueryRowContext(ctx, `
		INSERT INTO store_schedules (business_id, weekday, open_time, close_time, playlist_id, is_closed)
		VALUES ($1::uuid, $2, $3::time, $4::time, $5::uuid, $6)
		ON CONFLICT (business_id, weekday) DO UPDATE SET
			open_time = EXCLUDED.open_time, close_time = EXCLUDED.close_time,
			playlist_id = EXCLUDED.playlist_id, is_closed = EXCLUDED.is_closed,
			updated_at = NOW()
		RETURNING `+storeScheduleColumns,
		businessID, weekday, open, closeTime, nullStr(playlistID), isClosed))
	return s, err
}

// ListSchedulesByBusiness returns a business's weekly schedule.
func (r *StoreRepo) ListSchedulesByBusiness(ctx context.Context, businessID string) ([]*domain.StoreSchedule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+storeScheduleColumns+` FROM store_schedules
		WHERE business_id = $1::uuid ORDER BY weekday ASC`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.StoreSchedule{}
	for rows.Next() {
		s, err := scanStoreSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const storePlayLogColumns = `id::text, device_id::text, track_id::text, played_at,
	duration_played_secs, counted_as_play, created_at`

func scanStorePlayLog(row interface{ Scan(...any) error }) (*domain.StorePlayLog, error) {
	l := &domain.StorePlayLog{}
	err := row.Scan(&l.ID, &l.DeviceID, &l.TrackID, &l.PlayedAt,
		&l.DurationPlayedSecs, &l.CountedAsPlay, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// AppendPlayLog records one device play.
func (r *StoreRepo) AppendPlayLog(ctx context.Context, deviceID, trackID string, durationSecs int, counted bool) (*domain.StorePlayLog, error) {
	l, err := scanStorePlayLog(r.db.QueryRowContext(ctx, `
		INSERT INTO store_play_logs (device_id, track_id, duration_played_secs, counted_as_play)
		VALUES ($1::uuid, $2::uuid, $3, $4)
		RETURNING `+storePlayLogColumns, deviceID, trackID, durationSecs, counted))
	return l, err
}

// ListPlayLogsByDevice returns recent plays for a device.
func (r *StoreRepo) ListPlayLogsByDevice(ctx context.Context, deviceID string, limit int) ([]*domain.StorePlayLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+storePlayLogColumns+` FROM store_play_logs
		WHERE device_id = $1::uuid ORDER BY played_at DESC LIMIT $2`, deviceID, clampLimit(limit, 50, 200))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.StorePlayLog{}
	for rows.Next() {
		l, err := scanStorePlayLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CountPartnerPlaysInWindow counts counted store plays for a partner's tracks
// in [start, end) from APPROVED businesses' ACTIVE devices (WI-05 #21).
func (r *StoreRepo) CountPartnerPlaysInWindow(ctx context.Context, partnerID string, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM store_play_logs l
		JOIN store_devices d ON d.id = l.device_id
		JOIN store_businesses b ON b.id = d.business_id
		JOIN tracks t ON t.id = l.track_id
		WHERE t.partner_id = $1::uuid AND l.counted_as_play
			AND l.played_at >= $2 AND l.played_at < $3
			AND b.status = 'APPROVED' AND d.status = 'ACTIVE'`,
		partnerID, start, end).Scan(&n)
	return n, err
}

// SumQualifiedStoreSecs sums 30s+ store play seconds for a partner's tracks
// in [start, end) from APPROVED businesses' ACTIVE devices. This is the
// time-pro-rata basis of the 70:30 store pool (방향결정서 §5-1).
func (r *StoreRepo) SumQualifiedStoreSecs(ctx context.Context, partnerID string, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(l.duration_played_secs), 0)
		FROM store_play_logs l
		JOIN store_devices d ON d.id = l.device_id
		JOIN store_businesses b ON b.id = d.business_id
		JOIN tracks t ON t.id = l.track_id
		WHERE t.partner_id = $1::uuid AND l.counted_as_play
			AND l.played_at >= $2 AND l.played_at < $3
			AND b.status = 'APPROVED' AND d.status = 'ACTIVE'`,
		partnerID, start, end).Scan(&n)
	return n, err
}

// SumQualifiedStoreSecsAll sums the same basis across all partners: the
// denominator of the 70:30 store-pool split for the window.
func (r *StoreRepo) SumQualifiedStoreSecsAll(ctx context.Context, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(l.duration_played_secs), 0)
		FROM store_play_logs l
		JOIN store_devices d ON d.id = l.device_id
		JOIN store_businesses b ON b.id = d.business_id
		JOIN tracks t ON t.id = l.track_id
		WHERE l.counted_as_play
			AND l.played_at >= $1 AND l.played_at < $2
			AND b.status = 'APPROVED' AND d.status = 'ACTIVE'`,
		start, end).Scan(&n)
	return n, err
}

// CountCountedPlaysByBusiness counts counted store plays per track for a
// business since a time (settlement input, WI-05 #21).
func (r *StoreRepo) CountCountedPlaysByBusiness(ctx context.Context, businessID string, since time.Time) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.track_id::text, COUNT(*)
		FROM store_play_logs l
		JOIN store_devices d ON d.id = l.device_id
		WHERE d.business_id = $1::uuid AND l.counted_as_play AND l.played_at >= $2
		GROUP BY l.track_id`, businessID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var trackID string
		var n int64
		if err := rows.Scan(&trackID, &n); err != nil {
			return nil, err
		}
		out[trackID] = n
	}
	return out, rows.Err()
}
