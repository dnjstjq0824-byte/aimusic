package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/toss"
)

type fakeBillingStore struct {
	plans  map[string]*domain.BillingPlan
	orders map[string]*domain.BillingOrder
	subs   map[string]*domain.BillingSubscription
	events map[string]bool
}

func (f *fakeBillingStore) ListActivePlans(_ context.Context) ([]*domain.BillingPlan, error) {
	var out []*domain.BillingPlan
	for _, p := range f.plans {
		if p.IsActive {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeBillingStore) GetPlanByCode(_ context.Context, code string) (*domain.BillingPlan, error) {
	for _, p := range f.plans {
		if p.Code == code && p.IsActive {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeBillingStore) GetPlanByID(_ context.Context, id string) (*domain.BillingPlan, error) {
	p, ok := f.plans[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakeBillingStore) CreateOrder(_ context.Context, userID, planID, orderID string, amount int) (*domain.BillingOrder, error) {
	o := &domain.BillingOrder{ID: "ord-row-1", UserID: userID, PlanID: planID, OrderID: orderID, AmountKRW: amount, Status: domain.BillingPending}
	if f.orders == nil {
		f.orders = map[string]*domain.BillingOrder{}
	}
	f.orders[orderID] = o
	return o, nil
}

func (f *fakeBillingStore) GetOrderByMerchantID(_ context.Context, orderID string) (*domain.BillingOrder, error) {
	o, ok := f.orders[orderID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (f *fakeBillingStore) MarkOrderPaid(_ context.Context, orderID, key string) (*domain.BillingOrder, error) {
	o, ok := f.orders[orderID]
	if !ok || o.Status != domain.BillingPending {
		return nil, domain.ErrNotFound
	}
	o.Status = domain.BillingPaid
	o.TossPaymentKey = &key
	return o, nil
}

func (f *fakeBillingStore) SetOrderStatus(_ context.Context, orderID, status string) error {
	if o, ok := f.orders[orderID]; ok {
		o.Status = status
	}
	return nil
}

func (f *fakeBillingStore) ActivateSubscription(_ context.Context, userID, planID, orderID string, start, end time.Time) (*domain.BillingSubscription, error) {
	for _, s := range f.subs {
		if s.UserID == userID && s.Status == domain.BillSubActive {
			s.Status = domain.BillSubExpired
		}
	}
	s := &domain.BillingSubscription{ID: "sub-1", UserID: userID, PlanID: planID, OrderID: &orderID, Status: domain.BillSubActive, CurrentPeriodStart: start, CurrentPeriodEnd: end}
	if f.subs == nil {
		f.subs = map[string]*domain.BillingSubscription{}
	}
	f.subs[userID] = s
	return s, nil
}

func (f *fakeBillingStore) GetActiveSubscription(_ context.Context, userID string) (*domain.BillingSubscription, error) {
	s, ok := f.subs[userID]
	if !ok || s.Status != domain.BillSubActive {
		return nil, domain.ErrNotFound
	}
	return s, nil
}

func (f *fakeBillingStore) CancelActiveSubscription(_ context.Context, userID string) error {
	if s, ok := f.subs[userID]; ok && s.Status == domain.BillSubActive {
		s.Status = domain.BillSubCancelled
	}
	return nil
}

func (f *fakeBillingStore) ClaimWebhookEvent(_ context.Context, _, eventID, _ string) (bool, error) {
	if f.events == nil {
		f.events = map[string]bool{}
	}
	if f.events[eventID] {
		return false, nil
	}
	f.events[eventID] = true
	return true, nil
}

func (f *fakeBillingStore) MarkWebhookProcessed(_ context.Context, _, _ string) error { return nil }

// stubToss emulates confirm + payment query with scripted statuses.
func stubToss(t *testing.T, confirmStatus string, queryStatus map[string]string, hits *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/payments/confirm" {
			if confirmStatus == "ERROR" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"code":"INVALID_CARD","message":"bad card"}`))
				return
			}
			_, _ = w.Write([]byte(`{"paymentKey":"pk-1","orderId":"ORD","status":"` + confirmStatus + `","totalAmount":5900,"method":"CARD"}`))
			return
		}
		key := strings.TrimPrefix(r.URL.Path, "/v1/payments/")
		status := queryStatus[key]
		if status == "" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"no"}`))
			return
		}
		_, _ = w.Write([]byte(`{"paymentKey":"` + key + `","orderId":"ORD","status":"` + status + `","totalAmount":5900,"method":"CARD"}`))
	}))
}

func billingTestService(t *testing.T, store *fakeBillingStore, confirmStatus string, query map[string]string) (*BillingService, *int) {
	t.Helper()
	hits := 0
	srv := stubToss(t, confirmStatus, query, &hits)
	t.Cleanup(srv.Close)
	return &BillingService{
		billing: store, toss: toss.NewClient("secret", srv.URL, srv.Client()),
		enabled: true, now: func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) },
	}, &hits
}

func seedPlan() map[string]*domain.BillingPlan {
	return map[string]*domain.BillingPlan{
		"plan-1": {ID: "plan-1", Code: "LISTENER_PLUS_MONTHLY", NameKo: "리스너 플러스", AmountKRW: 5900, Interval: "MONTHLY", IsActive: true},
	}
}

func TestCreateOrderHappy(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	s, _ := billingTestService(t, store, "DONE", nil)
	order, plan, err := s.CreateOrder(context.Background(), "u-1", "LISTENER_PLUS_MONTHLY")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.HasPrefix(order.OrderID, "AVIBE-20261107-") || order.AmountKRW != 5900 || order.Status != domain.BillingPending {
		t.Fatalf("order = %+v", order)
	}
	if plan.Code != "LISTENER_PLUS_MONTHLY" {
		t.Fatalf("plan = %+v", plan)
	}
	if _, _, err := s.CreateOrder(context.Background(), "u-1", "NOPE"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("unknown plan err = %v", err)
	}
}

func TestCreateOrderDisabled(t *testing.T) {
	s := &BillingService{billing: &fakeBillingStore{plans: seedPlan()}, enabled: false, now: time.Now}
	if _, _, err := s.CreateOrder(context.Background(), "u-1", "LISTENER_PLUS_MONTHLY"); !errors.Is(err, domain.ErrPaymentUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if plans, err := s.Plans(context.Background()); err != nil || len(plans) != 1 {
		t.Fatalf("plans must work without PG keys: %v %d", err, len(plans))
	}
}

func TestConfirmActivatesSubscription(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	fixedNow := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PaymentKey string `json:"paymentKey"`
			OrderID    string `json:"orderId"`
			Amount     int    `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"` + req.PaymentKey + `","orderId":"` + req.OrderID +
			`","status":"DONE","totalAmount":5900,"method":"CARD"}`))
	}))
	t.Cleanup(srv.Close)
	s := &BillingService{billing: store, toss: toss.NewClient("s", srv.URL, srv.Client()), enabled: true, now: func() time.Time { return fixedNow }}
	ctx := context.Background()
	order, _, err := s.CreateOrder(ctx, "u-1", "LISTENER_PLUS_MONTHLY")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sub, err := s.Confirm(ctx, "u-1", "pk-1", order.OrderID, 5900)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if sub.Status != domain.BillSubActive || sub.Plan == nil || sub.Plan.Code != "LISTENER_PLUS_MONTHLY" {
		t.Fatalf("sub = %+v", sub)
	}
	if !sub.CurrentPeriodStart.Equal(fixedNow) || sub.CurrentPeriodEnd != fixedNow.AddDate(0, 1, 0) {
		t.Fatalf("period = %v .. %v", sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	}
	if store.orders[order.OrderID].Status != domain.BillingPaid {
		t.Fatalf("order status = %s", store.orders[order.OrderID].Status)
	}
	// Double confirm of a processed order must fail, not double-activate.
	if _, err := s.Confirm(ctx, "u-1", "pk-1", order.OrderID, 5900); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("double confirm err = %v", err)
	}
}

func TestConfirmRejectsMisuse(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	s, hits := billingTestService(t, store, "DONE", nil)
	ctx := context.Background()
	order, _, err := s.CreateOrder(ctx, "u-1", "LISTENER_PLUS_MONTHLY")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Confirm(ctx, "u-2", "pk-1", order.OrderID, 5900); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("foreign order err = %v", err)
	}
	if _, err := s.Confirm(ctx, "u-1", "pk-1", order.OrderID, 100); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("amount mismatch err = %v", err)
	}
	if *hits != 0 {
		t.Fatalf("toss called before local checks: %d", *hits)
	}
}

func TestConfirmTossFailureMarksFailed(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	s, _ := billingTestService(t, store, "ERROR", nil)
	ctx := context.Background()
	order, _, _ := s.CreateOrder(ctx, "u-1", "LISTENER_PLUS_MONTHLY")
	if _, err := s.Confirm(ctx, "u-1", "pk-1", order.OrderID, 5900); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("toss failure err = %v", err)
	}
	if store.orders[order.OrderID].Status != domain.BillingFailed {
		t.Fatalf("order status = %s", store.orders[order.OrderID].Status)
	}
}

func TestWebhookDoneAppliesOnce(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	store.orders = map[string]*domain.BillingOrder{
		"m-1": {ID: "row-1", UserID: "u-1", PlanID: "plan-1", OrderID: "m-1", AmountKRW: 5900, Status: domain.BillingPending},
	}
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"pk-1","orderId":"m-1","status":"DONE","totalAmount":5900,"method":"CARD"}`))
	}))
	t.Cleanup(srv.Close)
	s := &BillingService{billing: store, toss: toss.NewClient("s", srv.URL, srv.Client()), enabled: true, now: time.Now}
	raw := []byte(`{"eventType":"PAYMENT_STATUS_CHANGED","data":{"paymentKey":"pk-1","orderId":"m-1","status":"DONE"}}`)
	if err := s.HandleWebhook(context.Background(), raw); err != nil {
		t.Fatalf("webhook: %v", err)
	}
	if store.orders["m-1"].Status != domain.BillingPaid {
		t.Fatalf("order status = %s", store.orders["m-1"].Status)
	}
	sub, err := store.GetActiveSubscription(context.Background(), "u-1")
	if err != nil || sub.Status != domain.BillSubActive {
		t.Fatalf("subscription missing: %v", err)
	}
	if !sub.CurrentPeriodEnd.After(sub.CurrentPeriodStart) {
		t.Fatalf("period not set: %+v", sub)
	}
	// Duplicate delivery: acknowledged without re-query or re-apply.
	if err := s.HandleWebhook(context.Background(), raw); err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if hits != 1 {
		t.Fatalf("toss hits = %d, want 1", hits)
	}
}

func TestWebhookCanceledCancelsSubscription(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	store.orders = map[string]*domain.BillingOrder{
		"m-2": {ID: "row-2", UserID: "u-2", PlanID: "plan-1", OrderID: "m-2", AmountKRW: 5900, Status: domain.BillingPaid},
	}
	store.subs = map[string]*domain.BillingSubscription{
		"u-2": {ID: "sub-2", UserID: "u-2", PlanID: "plan-1", Status: domain.BillSubActive},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"pk-2","orderId":"m-2","status":"CANCELED","totalAmount":5900,"method":"CARD"}`))
	}))
	t.Cleanup(srv.Close)
	s := &BillingService{billing: store, toss: toss.NewClient("s", srv.URL, srv.Client()), enabled: true, now: time.Now}
	raw := []byte(`{"eventType":"PAYMENT_STATUS_CHANGED","data":{"paymentKey":"pk-2","orderId":"m-2","status":"CANCELED"}}`)
	if err := s.HandleWebhook(context.Background(), raw); err != nil {
		t.Fatalf("webhook: %v", err)
	}
	if store.orders["m-2"].Status != domain.BillingCancelled {
		t.Fatalf("order status = %s", store.orders["m-2"].Status)
	}
	if store.subs["u-2"].Status != domain.BillSubCancelled {
		t.Fatalf("sub status = %s", store.subs["u-2"].Status)
	}
}

func TestWebhookRejectsAmountMismatch(t *testing.T) {
	store := &fakeBillingStore{plans: seedPlan()}
	store.orders = map[string]*domain.BillingOrder{
		"m-3": {ID: "row-3", UserID: "u-3", PlanID: "plan-1", OrderID: "m-3", AmountKRW: 5900, Status: domain.BillingPending},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"pk-3","orderId":"m-3","status":"DONE","totalAmount":100,"method":"CARD"}`))
	}))
	t.Cleanup(srv.Close)
	s := &BillingService{billing: store, toss: toss.NewClient("s", srv.URL, srv.Client()), enabled: true, now: time.Now}
	raw := []byte(`{"eventType":"PAYMENT_STATUS_CHANGED","data":{"paymentKey":"pk-3","orderId":"m-3","status":"DONE"}}`)
	if err := s.HandleWebhook(context.Background(), raw); err == nil {
		t.Fatalf("mismatch accepted")
	}
	if store.orders["m-3"].Status != domain.BillingPending {
		t.Fatalf("order status = %s", store.orders["m-3"].Status)
	}
}
