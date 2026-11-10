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
const RefundTermsVersion = "2026-11-10-v1"

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
		if err := s.billing.SetOrderStatus(ctx, order.OrderID, domain.BillingCancelled); err != nil {
			return err
		}
		_ = s.billing.CancelActiveSubscription(ctx, order.UserID)
	case "PARTIAL_CANCELED":
		// Partial refunds keep the pass; the order stays PAID and the event
		// is journaled for finance review.
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

// CancelSubscription ends the user's ACTIVE pass (single-term passes do not
// auto-renew; this records intent and blocks further use immediately).
func (s *BillingService) CancelSubscription(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("%w: user", domain.ErrInvalidInput)
	}
	return s.billing.CancelActiveSubscription(ctx, userID)
}

// Refund voids the order behind the user's ACTIVE pass (empty orderRef
// resolves through the subscription) or a named merchant order.
func (s *BillingService) Refund(ctx context.Context, userID, orderRef, reason string) (*domain.BillingOrder, error) {
	merchantOrderID := strings.TrimSpace(orderRef)
	if merchantOrderID == "" {
		sub, err := s.billing.GetActiveSubscription(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("%w: no active pass to refund", domain.ErrInvalidInput)
		}
		if sub.OrderID == nil || *sub.OrderID == "" {
			return nil, fmt.Errorf("%w: pass has no order", domain.ErrInvalidInput)
		}
		orderRow, err := s.billing.GetOrder(ctx, *sub.OrderID)
		if err != nil {
			return nil, fmt.Errorf("%w: pass order missing", domain.ErrInvalidInput)
		}
		merchantOrderID = orderRow.OrderID
	}
	return s.refundMerchantOrder(ctx, userID, merchantOrderID, reason)
}

// refundMerchantOrder voids a PAID order at Toss in full, marks it REFUNDED,
// and ends the pass immediately. Already-cancelled-at-PG orders converge to
// REFUNDED so double submits and confirm/webhook races stay idempotent.
func (s *BillingService) refundMerchantOrder(ctx context.Context, userID, merchantOrderID, reason string) (*domain.BillingOrder, error) {
	if !s.enabled {
		return nil, domain.ErrPaymentUnavailable
	}
	if userID == "" || strings.TrimSpace(merchantOrderID) == "" {
		return nil, fmt.Errorf("%w: user/order", domain.ErrInvalidInput)
	}
	order, err := s.billing.GetOrderByMerchantID(ctx, merchantOrderID)
	if err != nil {
		return nil, fmt.Errorf("%w: unknown order", domain.ErrInvalidInput)
	}
	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if order.Status != domain.BillingPaid || order.TossPaymentKey == nil || *order.TossPaymentKey == "" {
		return nil, fmt.Errorf("%w: order is not refundable", domain.ErrInvalidInput)
	}
	if _, err := s.toss.CancelPayment(ctx, *order.TossPaymentKey, reason); err != nil {
		var te *toss.TossError
		if !(errors.As(err, &te) && te.Code == "ALREADY_CANCELED_TRANSACTION") {
			return nil, err
		}
	}
	if err := s.billing.SetOrderStatus(ctx, order.OrderID, domain.BillingRefunded); err != nil {
		return nil, err
	}
	_ = s.billing.CancelActiveSubscription(ctx, userID)
	return s.billing.GetOrderByMerchantID(ctx, merchantOrderID)
}
