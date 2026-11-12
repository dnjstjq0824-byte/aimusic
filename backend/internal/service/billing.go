package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/toss"
	"avibe-platform/backend/internal/repository"
)

// billingStore is the billing persistence surface.
type billingStore interface {
	ListActivePlans(ctx context.Context) ([]*domain.BillingPlan, error)
	GetPlanByCode(ctx context.Context, code string) (*domain.BillingPlan, error)
	GetPlanByID(ctx context.Context, id string) (*domain.BillingPlan, error)
	CreateOrder(ctx context.Context, userID, planID, orderID string, amountKRW int, consentAt time.Time, termsVersion string) (*domain.BillingOrder, error)
	GetOrderByMerchantID(ctx context.Context, orderID string) (*domain.BillingOrder, error)
	GetOrder(ctx context.Context, id string) (*domain.BillingOrder, error)
	MarkOrderPaid(ctx context.Context, orderID, paymentKey string) (*domain.BillingOrder, error)
	SetOrderStatus(ctx context.Context, orderID, status string) error
	ActivateSubscription(ctx context.Context, userID, planID, orderID string, periodStart, periodEnd time.Time) (*domain.BillingSubscription, error)
	GetActiveSubscription(ctx context.Context, userID string) (*domain.BillingSubscription, error)
	CancelActiveSubscription(ctx context.Context, userID string) error
	ClaimWebhookEvent(ctx context.Context, provider, eventID, payload string) (bool, error)
	FirstUsageSince(ctx context.Context, userID string, since time.Time) (*time.Time, error)
	RequestCancelAtPeriodEnd(ctx context.Context, userID string) error
	ClaimRefund(ctx context.Context, merchantOrderID string) (bool, error)
	ReleaseRefundClaim(ctx context.Context, merchantOrderID string) error
	RecordRefund(ctx context.Context, merchantOrderID, status string, amountKRW int) error
	MarkWebhookProcessed(ctx context.Context, provider, eventID string) error
}

// BillingService sells single-term plan passes through Toss Payments:
// order (PENDING) -> frontend Toss checkout -> confirm (PAID + ACTIVE
// subscription) with webhook verification by server-to-server re-query.
type BillingService struct {
	billing billingStore
	toss    *toss.Client
	enabled bool
	now     func() time.Time
}

// NewBillingService builds the service; empty TOSS_SECRET_KEY fails closed (503).
func NewBillingService(repos *repository.Repos, cfg *config.Config) *BillingService {
	return &BillingService{
		billing: repos.Billing,
		toss:    toss.NewClient(cfg.Toss.SecretKey, cfg.Toss.APIBaseURL, nil),
		enabled: strings.TrimSpace(cfg.Toss.SecretKey) != "",
		now:     time.Now,
	}
}

// Plans returns the purchasable plan catalog (works without PG keys).
func (s *BillingService) Plans(ctx context.Context) ([]*domain.BillingPlan, error) {
	return s.billing.ListActivePlans(ctx)
}

func merchantOrderID(now time.Time) (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("AVIBE-%s-%s", now.Format("20060102"), strings.ToUpper(hex.EncodeToString(b))), nil
}

// RefundTermsVersion is the checkout refund/withdrawal notice revision the
// buyer consents to (C6). Bump when the notice text changes.
const RefundTermsVersion = "2026-11-12-v2"

// CreateOrder opens a PENDING merchant order for a plan. The buyer must have
// consented to the refund/withdrawal notice on the checkout screen (C6): the
// consent timestamp is required and must be within the last 24 hours.
func (s *BillingService) CreateOrder(ctx context.Context, userID, planCode string, consentAt time.Time) (*domain.BillingOrder, *domain.BillingPlan, error) {
	if !s.enabled {
		return nil, nil, domain.ErrPaymentUnavailable
	}
	if userID == "" || strings.TrimSpace(planCode) == "" {
		return nil, nil, fmt.Errorf("%w: user/plan", domain.ErrInvalidInput)
	}
	if consentAt.IsZero() || s.now().Sub(consentAt) < 0 || s.now().Sub(consentAt) > 24*time.Hour {
		return nil, nil, fmt.Errorf("%w: 청약철회·환불 기준 동의가 필요합니다", domain.ErrInvalidInput)
	}
	if active, err := s.billing.GetActiveSubscription(ctx, userID); err == nil && active != nil {
		return nil, nil, domain.ErrActivePassExists
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}
	plan, err := s.billing.GetPlanByCode(ctx, strings.TrimSpace(planCode))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: unknown plan", domain.ErrInvalidInput)
	}
	orderID, err := merchantOrderID(s.now())
	if err != nil {
		return nil, nil, err
	}
	order, err := s.billing.CreateOrder(ctx, userID, plan.ID, orderID, plan.AmountKRW, consentAt, RefundTermsVersion)
	if err != nil {
		return nil, nil, err
	}
	return order, plan, nil
}

// Confirm approves a Toss payment and activates the subscription period.
func (s *BillingService) Confirm(ctx context.Context, userID, paymentKey, orderID string, amount int) (*domain.BillingSubscription, error) {
	if !s.enabled {
		return nil, domain.ErrPaymentUnavailable
	}
	if strings.TrimSpace(paymentKey) == "" || strings.TrimSpace(orderID) == "" || amount <= 0 {
		return nil, fmt.Errorf("%w: paymentKey/orderId/amount", domain.ErrInvalidInput)
	}
	order, err := s.billing.GetOrderByMerchantID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: unknown order", domain.ErrInvalidInput)
	}
	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if order.Status != domain.BillingPending {
		return nil, fmt.Errorf("%w: order already processed", domain.ErrInvalidInput)
	}
	if order.AmountKRW != amount {
		return nil, fmt.Errorf("%w: amount mismatch", domain.ErrInvalidInput)
	}
	p, err := s.toss.ConfirmPayment(ctx, paymentKey, orderID, amount)
	if err != nil {
		_ = s.billing.SetOrderStatus(ctx, orderID, domain.BillingFailed)
		var te *toss.TossError
		if errors.As(err, &te) {
			return nil, fmt.Errorf("%w: toss %s", domain.ErrInvalidInput, te.Code)
		}
		return nil, err
	}
	if p.OrderID != orderID || p.TotalAmount != amount {
		_ = s.billing.SetOrderStatus(ctx, orderID, domain.BillingFailed)
		return nil, fmt.Errorf("%w: toss verification mismatch", domain.ErrInvalidInput)
	}
	if _, err := s.billing.MarkOrderPaid(ctx, orderID, paymentKey); err != nil {
		// A raced webhook already applied this order: return the live subscription.
		sub, subErr := s.billing.GetActiveSubscription(ctx, userID)
		if subErr != nil {
			return nil, subErr
		}
		return s.withPlan(ctx, sub)
	}
	start := s.now()
	sub, err := s.billing.ActivateSubscription(ctx, userID, order.PlanID, order.ID, start, start.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}
	return s.withPlan(ctx, sub)
}

func (s *BillingService) withPlan(ctx context.Context, sub *domain.BillingSubscription) (*domain.BillingSubscription, error) {
	plan, err := s.billing.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	sub.Plan = plan
	return sub, nil
}

// tossWebhookEvent is the Toss transmission envelope subset billing needs.
type tossWebhookEvent struct {
	EventType string `json:"eventType"`
	Data      struct {
		PaymentKey string `json:"paymentKey"`
		OrderID    string `json:"orderId"`
		Status     string `json:"status"`
	} `json:"data"`
}

// HandleWebhook verifies a Toss event by re-querying the payment and applies
// it idempotently. Duplicate deliveries are acknowledged without re-applying.
func (s *BillingService) HandleWebhook(ctx context.Context, raw []byte) error {
	if !s.enabled {
		return domain.ErrPaymentUnavailable
	}
	var ev tossWebhookEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return fmt.Errorf("%w: bad webhook payload", domain.ErrInvalidInput)
	}
	if ev.Data.PaymentKey == "" || ev.Data.OrderID == "" {
		return fmt.Errorf("%w: paymentKey/orderId", domain.ErrInvalidInput)
	}
	eventID := ev.EventType + "|" + ev.Data.PaymentKey + "|" + ev.Data.OrderID + "|" + ev.Data.Status
	fresh, err := s.billing.ClaimWebhookEvent(ctx, "TOSS", eventID, string(raw))
	if err != nil {
		return err
	}
	if !fresh {
		return nil
	}
	p, err := s.toss.GetPayment(ctx, ev.Data.PaymentKey)
	if err != nil {
		return err
	}
	order, err := s.billing.GetOrderByMerchantID(ctx, ev.Data.OrderID)
	if err != nil {
		return fmt.Errorf("%w: unknown order", domain.ErrInvalidInput)
	}
	if p.OrderID != order.OrderID || p.TotalAmount != order.AmountKRW {
		return fmt.Errorf("%w: toss verification mismatch", domain.ErrInvalidInput)
	}
	switch p.Status {
	case "DONE":
		paid, err := s.billing.MarkOrderPaid(ctx, order.OrderID, p.PaymentKey)
		if err != nil {
			break // already applied (confirm raced ahead); still acknowledge
		}
		_ = paid
		start := s.now()
		if _, err := s.billing.ActivateSubscription(ctx, order.UserID, order.PlanID, order.ID, start, start.AddDate(0, 1, 0)); err != nil {
			return err
		}
	case "CANCELED":
		if order.Status != domain.BillingRefunded && order.Status != domain.BillingPartiallyRefunded {
			if err := s.billing.SetOrderStatus(ctx, order.OrderID, domain.BillingCancelled); err != nil {
				return err
			}
		}
		_ = s.billing.CancelActiveSubscription(ctx, order.UserID)
	case "PARTIAL_CANCELED":
		// Our own prorated refunds already ended the pass and set
		// PARTIALLY_REFUNDED; this event never revives it. Console-issued
		// partial cancels are journaled for finance review.
	default:
		// Unknown/future statuses are journaled but not applied.
	}
	return s.billing.MarkWebhookProcessed(ctx, "TOSS", eventID)
}

// MySubscription returns the user's ACTIVE subscription with its plan, or
// (nil, nil) when the user holds no active pass.
func (s *BillingService) MySubscription(ctx context.Context, userID string) (*domain.BillingSubscription, error) {
	sub, err := s.billing.GetActiveSubscription(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return s.withPlan(ctx, sub)
}

// Cancellation modes (2026-11-12 refund policy, same structure as Korean
// music streaming): the customer must pick one explicitly.
const (
	// CancelEndOfPeriod (일반해지): keep the pass until it expires, no refund.
	CancelEndOfPeriod = "END_OF_PERIOD"
	// CancelImmediate (청약철회 or 중도해지): end now, refund per RefundQuote.
	CancelImmediate = "IMMEDIATE"
)

// Refund quote types.
const (
	// RefundWithdrawal (청약철회): within 7 days of payment and unused, full refund.
	RefundWithdrawal = "WITHDRAWAL"
	// RefundProrated (중도해지): payment minus the used days, no penalty fee.
	RefundProrated = "PRORATED"
)

// WithdrawalWindow is the 청약철회 window after payment.
const WithdrawalWindow = 7 * 24 * time.Hour

// RefundQuote is the server-computed refund for ending the active pass now.
type RefundQuote struct {
	Type            string     `json:"type"`
	AmountKRW       int        `json:"amountKrw"`
	RefundKRW       int        `json:"refundKrw"`
	UsedDays        int        `json:"usedDays"`
	TotalDays       int        `json:"totalDays"`
	PaidAt          time.Time  `json:"paidAt"`
	PeriodEnd       time.Time  `json:"periodEnd"`
	HasUsage        bool       `json:"hasUsage"`
	CancelRequested bool       `json:"cancelRequested"`
	FirstUsageAt    *time.Time `json:"firstUsageAt,omitempty"`
}

// ComputeRefund applies the refund policy. used/total are whole days
// (used rounded up, at least 1); the deduction rounds up to the won.
func ComputeRefund(amount int, paidAt, periodStart, periodEnd, now time.Time, hasUsage bool) RefundQuote {
	day := 24 * time.Hour
	total := int(periodEnd.Sub(periodStart).Round(day) / day)
	if total < 1 {
		total = 1
	}
	elapsed := now.Sub(periodStart)
	if elapsed < 0 {
		elapsed = 0
	}
	used := int(elapsed / day)
	if elapsed%day > 0 || used == 0 {
		used++
	}
	if used > total {
		used = total
	}
	q := RefundQuote{AmountKRW: amount, UsedDays: used, TotalDays: total, PaidAt: paidAt, PeriodEnd: periodEnd, HasUsage: hasUsage}
	if !hasUsage && now.Sub(paidAt) < WithdrawalWindow {
		q.Type, q.RefundKRW = RefundWithdrawal, amount
		return q
	}
	deduct := (amount*used + total - 1) / total
	refund := amount - deduct
	if refund < 0 {
		refund = 0
	}
	q.Type, q.RefundKRW = RefundProrated, refund
	return q
}

// activePass loads the caller's active pass and its paid order.
func (s *BillingService) activePass(ctx context.Context, userID string) (*domain.BillingSubscription, *domain.BillingOrder, error) {
	if userID == "" {
		return nil, nil, fmt.Errorf("%w: user", domain.ErrInvalidInput)
	}
	sub, err := s.billing.GetActiveSubscription(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: 이용 중인 이용권이 없습니다", domain.ErrInvalidInput)
		}
		return nil, nil, err
	}
	if sub.OrderID == nil || *sub.OrderID == "" {
		return nil, nil, fmt.Errorf("%w: pass has no order", domain.ErrInvalidInput)
	}
	order, err := s.billing.GetOrder(ctx, *sub.OrderID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: pass order missing", domain.ErrInvalidInput)
	}
	return sub, order, nil
}

// QuoteRefund returns what ending the active pass now would refund.
func (s *BillingService) QuoteRefund(ctx context.Context, userID string) (*RefundQuote, error) {
	sub, order, err := s.activePass(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.quote(ctx, sub, order)
}

func (s *BillingService) quote(ctx context.Context, sub *domain.BillingSubscription, order *domain.BillingOrder) (*RefundQuote, error) {
	paidAt := sub.CurrentPeriodStart
	if order.ApprovedAt != nil {
		paidAt = *order.ApprovedAt
	}
	first, err := s.billing.FirstUsageSince(ctx, sub.UserID, sub.CurrentPeriodStart)
	if err != nil {
		return nil, err
	}
	q := ComputeRefund(order.AmountKRW, paidAt, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, s.now(), first != nil)
	q.FirstUsageAt = first
	q.CancelRequested = sub.CancelRequestedAt != nil
	return &q, nil
}

// CancelResult reports what a cancellation did.
type CancelResult struct {
	Mode         string                      `json:"mode"`
	Quote        *RefundQuote                `json:"quote,omitempty"`
	Order        *domain.BillingOrder        `json:"order,omitempty"`
	Subscription *domain.BillingSubscription `json:"subscription,omitempty"`
}

// Cancel ends the caller's pass in the chosen mode. END_OF_PERIOD records
// the request and keeps access until expiry; IMMEDIATE refunds per the
// quote (full within the withdrawal window, otherwise prorated) and ends
// the pass now. The refund is claimed on the order row first, so a double
// submit can never refund twice.
func (s *BillingService) Cancel(ctx context.Context, userID, mode, reason string) (*CancelResult, error) {
	switch mode {
	case CancelEndOfPeriod:
		sub, _, err := s.activePass(ctx, userID)
		if err != nil {
			return nil, err
		}
		if err := s.billing.RequestCancelAtPeriodEnd(ctx, userID); err != nil {
			return nil, err
		}
		if sub.CancelRequestedAt == nil {
			now := s.now()
			sub.CancelRequestedAt = &now
		}
		return &CancelResult{Mode: mode, Subscription: sub}, nil
	case CancelImmediate:
		return s.cancelImmediate(ctx, userID, reason)
	default:
		return nil, fmt.Errorf("%w: mode must be END_OF_PERIOD or IMMEDIATE", domain.ErrInvalidInput)
	}
}

func (s *BillingService) cancelImmediate(ctx context.Context, userID, reason string) (*CancelResult, error) {
	if !s.enabled {
		return nil, domain.ErrPaymentUnavailable
	}
	sub, order, err := s.activePass(ctx, userID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}
	q, err := s.quote(ctx, sub, order)
	if err != nil {
		return nil, err
	}
	if q.RefundKRW > 0 {
		if order.Status != domain.BillingPaid || order.TossPaymentKey == nil || *order.TossPaymentKey == "" {
			return nil, fmt.Errorf("%w: order is not refundable", domain.ErrInvalidInput)
		}
		claimed, err := s.billing.ClaimRefund(ctx, order.OrderID)
		if err != nil {
			return nil, err
		}
		if !claimed {
			return nil, fmt.Errorf("%w: 이미 환불 처리 중이거나 완료된 주문입니다", domain.ErrInvalidInput)
		}
		cancelAmount := q.RefundKRW
		if cancelAmount == order.AmountKRW {
			cancelAmount = 0 // full cancel
		}
		if _, err := s.toss.CancelPaymentAmount(ctx, *order.TossPaymentKey, reason, cancelAmount); err != nil {
			var te *toss.TossError
			if !(errors.As(err, &te) && te.Code == "ALREADY_CANCELED_TRANSACTION") {
				_ = s.billing.ReleaseRefundClaim(ctx, order.OrderID)
				return nil, err
			}
		}
		status := domain.BillingRefunded
		if q.RefundKRW < order.AmountKRW {
			status = domain.BillingPartiallyRefunded
		}
		if err := s.billing.RecordRefund(ctx, order.OrderID, status, q.RefundKRW); err != nil {
			return nil, err
		}
	}
	if err := s.billing.CancelActiveSubscription(ctx, userID); err != nil {
		return nil, err
	}
	updated, err := s.billing.GetOrderByMerchantID(ctx, order.OrderID)
	if err != nil {
		return nil, err
	}
	return &CancelResult{Mode: CancelImmediate, Quote: q, Order: updated}, nil
}

// CancelSubscription is the legacy no-mode cancel. Cancelling now needs an
// explicit mode, so it maps to END_OF_PERIOD (no refund, keep access).
func (s *BillingService) CancelSubscription(ctx context.Context, userID string) error {
	_, err := s.Cancel(ctx, userID, CancelEndOfPeriod, "")
	return err
}

// Refund ends the active pass immediately under the refund policy. A named
// order must be the active pass's order; there is no unconditional full
// refund path any more.
func (s *BillingService) Refund(ctx context.Context, userID, orderRef, reason string) (*domain.BillingOrder, error) {
	if !s.enabled {
		return nil, domain.ErrPaymentUnavailable
	}
	if ref := strings.TrimSpace(orderRef); ref != "" {
		named, err := s.billing.GetOrderByMerchantID(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("%w: unknown order", domain.ErrInvalidInput)
		}
		if named.UserID != userID {
			return nil, domain.ErrForbidden
		}
		_, active, err := s.activePass(ctx, userID)
		if err != nil {
			return nil, err
		}
		if active.OrderID != named.OrderID {
			return nil, fmt.Errorf("%w: only the active pass can be refunded", domain.ErrInvalidInput)
		}
	}
	res, err := s.Cancel(ctx, userID, CancelImmediate, reason)
	if err != nil {
		return nil, err
	}
	return res.Order, nil
}
