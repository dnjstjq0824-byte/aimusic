package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// BillingRepo accesses billing plans, orders, subscriptions, and the webhook journal.
type BillingRepo struct{ db *sql.DB }

func scanPlan(row interface{ Scan(...any) error }) (*domain.BillingPlan, error) {
	p := &domain.BillingPlan{}
	err := row.Scan(&p.ID, &p.Code, &p.NameKo, &p.AmountKRW, &p.Interval, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

const planColumns = `id::text, code, name_ko, amount_krw, interval, is_active, created_at, updated_at`

// ListActivePlans returns purchasable plans, cheapest first.
func (r *BillingRepo) ListActivePlans(ctx context.Context) ([]*domain.BillingPlan, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+planColumns+`
		FROM billing_plans WHERE is_active = TRUE ORDER BY amount_krw ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.BillingPlan{}
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPlanByCode returns one active plan by code.
func (r *BillingRepo) GetPlanByCode(ctx context.Context, code string) (*domain.BillingPlan, error) {
	p, err := scanPlan(r.db.QueryRowContext(ctx,
		`SELECT `+planColumns+` FROM billing_plans WHERE code = $1 AND is_active = TRUE`, code))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// GetPlanByID returns one plan by ID.
func (r *BillingRepo) GetPlanByID(ctx context.Context, id string) (*domain.BillingPlan, error) {
	p, err := scanPlan(r.db.QueryRowContext(ctx,
		`SELECT `+planColumns+` FROM billing_plans WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

func scanOrder(row interface{ Scan(...any) error }) (*domain.BillingOrder, error) {
	o := &domain.BillingOrder{}
	var key, termsVer sql.NullString
	var approved, consentAt, refundedAt sql.NullTime
	var refunded sql.NullInt64
	err := row.Scan(&o.ID, &o.UserID, &o.PlanID, &o.OrderID, &o.AmountKRW, &o.Status,
		&key, &approved, &consentAt, &termsVer, &refunded, &refundedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if refunded.Valid {
		v := int(refunded.Int64)
		o.RefundedAmountKRW = &v
	}
	o.RefundedAt = timePtr(refundedAt)
	o.TossPaymentKey = strPtr(key)
	o.ApprovedAt = timePtr(approved)
	o.RefundConsentAt = timePtr(consentAt)
	o.RefundTermsVersion = strPtr(termsVer)
	return o, nil
}

const orderColumns = `id::text, user_id::text, plan_id::text, order_id, amount_krw, status,
	toss_payment_key, approved_at, refund_consent_at, refund_terms_version,
	refunded_amount_krw, refunded_at, created_at, updated_at`

// CreateOrder inserts a PENDING merchant order with refund-consent evidence.
func (r *BillingRepo) CreateOrder(ctx context.Context, userID, planID, orderID string, amountKRW int, consentAt time.Time, termsVersion string) (*domain.BillingOrder, error) {
	return scanOrder(r.db.QueryRowContext(ctx, `
		INSERT INTO billing_orders (user_id, plan_id, order_id, amount_krw, refund_consent_at, refund_terms_version)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		RETURNING `+orderColumns, userID, planID, orderID, amountKRW, consentAt, nullStr(&termsVersion)))
}

// GetOrderByMerchantID returns an order by our merchant order ID.
func (r *BillingRepo) GetOrderByMerchantID(ctx context.Context, orderID string) (*domain.BillingOrder, error) {
	o, err := scanOrder(r.db.QueryRowContext(ctx,
		`SELECT `+orderColumns+` FROM billing_orders WHERE order_id = $1`, orderID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

// ListOrdersByUser returns a user's orders, newest first (data export).
func (r *BillingRepo) ListOrdersByUser(ctx context.Context, userID string, limit int) ([]*domain.BillingOrder, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+orderColumns+` FROM billing_orders
		WHERE user_id = $1::uuid ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.BillingOrder{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetOrder returns an order by its row ID.
func (r *BillingRepo) GetOrder(ctx context.Context, id string) (*domain.BillingOrder, error) {
	o, err := scanOrder(r.db.QueryRowContext(ctx,
		`SELECT `+orderColumns+` FROM billing_orders WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

// MarkOrderPaid transitions PENDING -> PAID exactly once (concurrency-safe:
// confirm and webhook racing cannot double-apply).
func (r *BillingRepo) MarkOrderPaid(ctx context.Context, orderID, paymentKey string) (*domain.BillingOrder, error) {
	o, err := scanOrder(r.db.QueryRowContext(ctx, `
		UPDATE billing_orders
		SET status = 'PAID', toss_payment_key = $2, approved_at = NOW()
		WHERE order_id = $1 AND status = 'PENDING'
		RETURNING `+orderColumns, orderID, paymentKey))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

// SetOrderStatus force-sets an order status (failed/cancelled/refunded paths).
func (r *BillingRepo) SetOrderStatus(ctx context.Context, orderID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE billing_orders SET status = $2 WHERE order_id = $1`, orderID, status)
	return err
}

func scanBillingSubscription(row interface{ Scan(...any) error }) (*domain.BillingSubscription, error) {
	s := &domain.BillingSubscription{}
	var order sql.NullString
	var cancelReq sql.NullTime
	err := row.Scan(&s.ID, &s.UserID, &s.PlanID, &order, &s.Status,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &cancelReq, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.OrderID = strPtr(order)
	s.CancelRequestedAt = timePtr(cancelReq)
	return s, nil
}

const billingSubscriptionColumns = `id::text, user_id::text, plan_id::text, order_id::text, status,
	current_period_start, current_period_end, cancel_requested_at, created_at, updated_at`

// ActivateSubscription replaces the user's ACTIVE subscription with a fresh period.
func (r *BillingRepo) ActivateSubscription(ctx context.Context, userID, planID, orderID string, periodStart, periodEnd time.Time) (*domain.BillingSubscription, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE billing_subscriptions SET status = 'EXPIRED' WHERE user_id = $1::uuid AND status = 'ACTIVE'`,
		userID); err != nil {
		return nil, err
	}
	s, err := scanBillingSubscription(tx.QueryRowContext(ctx, `
		INSERT INTO billing_subscriptions (user_id, plan_id, order_id, status, current_period_start, current_period_end)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'ACTIVE', $4, $5)
		RETURNING `+billingSubscriptionColumns, userID, planID, orderID, periodStart, periodEnd))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}

// GetActiveSubscription returns the user's in-period ACTIVE subscription, if
// any. Expired rows do not count: without the period predicate an expired
// pass would grant premium features forever.
func (r *BillingRepo) GetActiveSubscription(ctx context.Context, userID string) (*domain.BillingSubscription, error) {
	s, err := scanBillingSubscription(r.db.QueryRowContext(ctx,
		`SELECT `+billingSubscriptionColumns+` FROM billing_subscriptions
		WHERE user_id = $1::uuid AND status = 'ACTIVE' AND current_period_end > NOW()`, userID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return s, err
}

// CancelActiveSubscription moves the user's ACTIVE subscription to CANCELLED.
func (r *BillingRepo) CancelActiveSubscription(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE billing_subscriptions SET status = 'CANCELLED'
		WHERE user_id = $1::uuid AND status = 'ACTIVE'`, userID)
	return err
}

// ClaimWebhookEvent inserts a webhook journal row; ok=false means a duplicate
// delivery that must be acknowledged without re-applying.
func (r *BillingRepo) ClaimWebhookEvent(ctx context.Context, provider, eventID, payload string) (ok bool, err error) {
	// An event that was received but never finished processing (processed =
	// FALSE) is claimable again, so a Toss retry after a failure is applied
	// instead of being acknowledged as a duplicate (B-02).
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO billing_webhook_events (provider, event_id, payload)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (provider, event_id) DO UPDATE SET payload = EXCLUDED.payload
		WHERE billing_webhook_events.processed = FALSE`, provider, eventID, payload)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// MarkWebhookProcessed flags a webhook event as applied.
func (r *BillingRepo) MarkWebhookProcessed(ctx context.Context, provider, eventID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE billing_webhook_events SET processed = TRUE
		WHERE provider = $1 AND event_id = $2`, provider, eventID)
	return err
}

// FirstUsageSince returns the earliest pass usage (a counted stream or a
// download) at or after since, or nil when the pass is unused.
func (r *BillingRepo) FirstUsageSince(ctx context.Context, userID string, since time.Time) (*time.Time, error) {
	var first sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT LEAST(
			(SELECT MIN(played_at) FROM streaming_logs
			 WHERE user_id = $1::uuid AND counted_as_play AND played_at >= $2),
			(SELECT MIN(created_at) FROM download_logs
			 WHERE user_id = $1::uuid AND created_at >= $2))`, userID, since).Scan(&first)
	if err != nil {
		return nil, err
	}
	return timePtr(first), nil
}

// RequestCancelAtPeriodEnd records 일반해지 on the active pass.
func (r *BillingRepo) RequestCancelAtPeriodEnd(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE billing_subscriptions SET cancel_requested_at = COALESCE(cancel_requested_at, NOW())
		WHERE user_id = $1::uuid AND status = 'ACTIVE'`, userID)
	return err
}

// ClaimRefund atomically reserves a PAID order for one refund; false means
// another request already claimed or refunded it.
func (r *BillingRepo) ClaimRefund(ctx context.Context, merchantOrderID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE billing_orders SET refunded_at = NOW()
		WHERE order_id = $1 AND status = 'PAID' AND refunded_at IS NULL`, merchantOrderID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ReleaseRefundClaim undoes ClaimRefund after a failed PG cancel.
func (r *BillingRepo) ReleaseRefundClaim(ctx context.Context, merchantOrderID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE billing_orders SET refunded_at = NULL
		WHERE order_id = $1 AND status = 'PAID'`, merchantOrderID)
	return err
}

// RecordRefund finalizes a claimed refund.
func (r *BillingRepo) RecordRefund(ctx context.Context, merchantOrderID, status string, amountKRW int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE billing_orders SET status = $2, refunded_amount_krw = $3,
			refunded_at = COALESCE(refunded_at, NOW())
		WHERE order_id = $1`, merchantOrderID, status, amountKRW)
	return err
}
