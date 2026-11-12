package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/toss"
)

func TestComputeRefundPolicy(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	cases := []struct {
		name       string
		now        time.Time
		used       bool
		wantType   string
		wantRefund int
	}{
		{"6d23h unused -> full", start.Add(6*24*time.Hour + 23*time.Hour), false, RefundWithdrawal, 5900},
		{"7d unused -> prorated", start.Add(7 * 24 * time.Hour), false, RefundProrated, 5900 - (5900*7+29)/30},
		{"day 1 used -> prorated", start.Add(2 * time.Hour), true, RefundProrated, 5900 - (5900*1+29)/30},
		{"10 of 30 days used", start.Add(10 * 24 * time.Hour), true, RefundProrated, 3933},
		{"last minute -> 0", end.Add(-time.Minute), true, RefundProrated, 0},
	}
	for _, c := range cases {
		q := ComputeRefund(5900, start, start, end, c.now, c.used)
		if q.Type != c.wantType || q.RefundKRW != c.wantRefund {
			t.Errorf("%s: got %s %d, want %s %d (used %d/%d)", c.name, q.Type, q.RefundKRW, c.wantType, c.wantRefund, q.UsedDays, q.TotalDays)
		}
	}
	// 10 of 30 days: deduction 1,967 won, refund 3,933 won (work order example).
	q := ComputeRefund(5900, start, start, end, start.Add(10*24*time.Hour), true)
	if q.AmountKRW-q.RefundKRW != 1967 {
		t.Fatalf("deduction = %d, want 1967", q.AmountKRW-q.RefundKRW)
	}
}

// cancelHarness runs the billing service against a Toss stub that records
// every cancel call.
type cancelHarness struct {
	svc     *BillingService
	store   *fakeBillingStore
	cancels []map[string]any
}

func newCancelHarness(t *testing.T, now time.Time, usage *time.Time) *cancelHarness {
	t.Helper()
	h := &cancelHarness{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		_ = json.Unmarshal(raw, &body)
		h.cancels = append(h.cancels, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"pk-c","orderId":"m-c","status":"CANCELED","totalAmount":5900,"method":"CARD"}`))
	}))
	t.Cleanup(srv.Close)
	pk := "pk-c"
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	rowID := "row-c"
	h.store = &fakeBillingStore{plans: seedPlan(), usage: usage}
	h.store.orders = map[string]*domain.BillingOrder{
		"m-c": {ID: rowID, UserID: "u-c", PlanID: "plan-1", OrderID: "m-c", AmountKRW: 5900, Status: domain.BillingPaid, TossPaymentKey: &pk, ApprovedAt: &start},
	}
	h.store.subs = map[string]*domain.BillingSubscription{
		"u-c": {ID: "sub-c", UserID: "u-c", PlanID: "plan-1", OrderID: &rowID, Status: domain.BillSubActive,
			CurrentPeriodStart: start, CurrentPeriodEnd: start.AddDate(0, 0, 30)},
	}
	h.svc = &BillingService{billing: h.store, toss: toss.NewClient("s", srv.URL, srv.Client()), enabled: true,
		now: func() time.Time { return now }}
	return h
}

func TestCancelImmediateProratedSendsPartialAmount(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	used := start.Add(time.Hour)
	h := newCancelHarness(t, start.Add(10*24*time.Hour), &used)
	res, err := h.svc.Cancel(context.Background(), "u-c", CancelImmediate, "")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if res.Quote.Type != RefundProrated || res.Quote.RefundKRW != 3933 {
		t.Fatalf("quote = %+v", res.Quote)
	}
	if len(h.cancels) != 1 || h.cancels[0]["cancelAmount"] != float64(3933) {
		t.Fatalf("toss cancels = %+v", h.cancels)
	}
	if o := h.store.orders["m-c"]; o.Status != domain.BillingPartiallyRefunded || *o.RefundedAmountKRW != 3933 {
		t.Fatalf("order = %+v", o)
	}
	if h.store.subs["u-c"].Status != domain.BillSubCancelled {
		t.Fatalf("pass not ended: %s", h.store.subs["u-c"].Status)
	}
	// A double submit never reaches Toss again.
	if _, err := h.svc.Cancel(context.Background(), "u-c", CancelImmediate, ""); err == nil {
		t.Fatal("second immediate cancel accepted")
	}
	if len(h.cancels) != 1 {
		t.Fatalf("toss called %d times", len(h.cancels))
	}
}

func TestCancelImmediateWithdrawalIsFullCancel(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	h := newCancelHarness(t, start.Add(3*24*time.Hour), nil)
	res, err := h.svc.Cancel(context.Background(), "u-c", CancelImmediate, "")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if res.Quote.Type != RefundWithdrawal || res.Quote.RefundKRW != 5900 {
		t.Fatalf("quote = %+v", res.Quote)
	}
	if _, has := h.cancels[0]["cancelAmount"]; has {
		t.Fatalf("full cancel must omit cancelAmount: %+v", h.cancels[0])
	}
	if h.store.orders["m-c"].Status != domain.BillingRefunded {
		t.Fatalf("status = %s", h.store.orders["m-c"].Status)
	}
}

func TestCancelImmediateZeroRefundSkipsToss(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	used := start.Add(time.Hour)
	h := newCancelHarness(t, start.AddDate(0, 0, 30).Add(-time.Minute), &used)
	res, err := h.svc.Cancel(context.Background(), "u-c", CancelImmediate, "")
	if err != nil || res.Quote.RefundKRW != 0 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(h.cancels) != 0 {
		t.Fatalf("toss called for 0 won refund")
	}
	if h.store.subs["u-c"].Status != domain.BillSubCancelled {
		t.Fatal("pass must end even with 0 won refund")
	}
}

func TestCancelEndOfPeriodKeepsAccess(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	h := newCancelHarness(t, start.Add(5*24*time.Hour), nil)
	res, err := h.svc.Cancel(context.Background(), "u-c", CancelEndOfPeriod, "")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	sub := h.store.subs["u-c"]
	if sub.Status != domain.BillSubActive || sub.CancelRequestedAt == nil || res.Subscription.CancelRequestedAt == nil {
		t.Fatalf("sub = %+v", sub)
	}
	if len(h.cancels) != 0 {
		t.Fatal("end-of-period cancel must not refund")
	}
	q, err := h.svc.QuoteRefund(context.Background(), "u-c")
	if err != nil || !q.CancelRequested {
		t.Fatalf("quote after request = %+v %v", q, err)
	}
	if _, err := h.svc.Cancel(context.Background(), "u-c", "", ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing mode err = %v", err)
	}
}

func TestCreateOrderBlockedWhileActivePass(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	h := newCancelHarness(t, start.Add(time.Hour), nil)
	if _, _, err := h.svc.CreateOrder(context.Background(), "u-c", "LISTENER_PLUS_MONTHLY", start.Add(time.Hour)); !errors.Is(err, domain.ErrActivePassExists) {
		t.Fatalf("repurchase err = %v", err)
	}
}

func TestWebhookRetryAfterFailureIsApplied(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	store.orders = map[string]*domain.BillingOrder{
		"ORD": {ID: "row-w", UserID: "u-w", PlanID: "plan-1", OrderID: "ORD", AmountKRW: 5900, Status: domain.BillingPending},
	}
	fail := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"FAILED_INTERNAL_SYSTEM_PROCESSING","message":"x"}`))
			return
		}
		_, _ = w.Write([]byte(`{"paymentKey":"pk-w","orderId":"ORD","status":"DONE","totalAmount":5900,"method":"CARD"}`))
	}))
	t.Cleanup(srv.Close)
	s := &BillingService{billing: store, toss: toss.NewClient("s", srv.URL, srv.Client()), enabled: true, now: time.Now}
	raw := []byte(`{"eventType":"PAYMENT_STATUS_CHANGED","data":{"paymentKey":"pk-w","orderId":"ORD","status":"DONE"}}`)
	if err := s.HandleWebhook(context.Background(), raw); err == nil {
		t.Fatal("first delivery should fail while Toss is down")
	}
	fail = false
	if err := s.HandleWebhook(context.Background(), raw); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if store.orders["ORD"].Status != domain.BillingPaid || store.subs["u-w"] == nil {
		t.Fatalf("retry not applied: order=%s sub=%v", store.orders["ORD"].Status, store.subs["u-w"])
	}
	// A third delivery after success is acknowledged without re-applying.
	before := store.subs["u-w"]
	if err := s.HandleWebhook(context.Background(), raw); err != nil || store.subs["u-w"] != before {
		t.Fatalf("processed event re-applied: err=%v", err)
	}
}
