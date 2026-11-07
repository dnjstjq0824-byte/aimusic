package toss

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func stubServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, NewClient("test-secret", srv.URL, srv.Client())
}

func TestConfirmPaymentSuccess(t *testing.T) {
	_, c := stubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/payments/confirm" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Errorf("missing basic auth")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["paymentKey"] != "pk-1" || body["orderId"] != "ord-1" || body["amount"] != float64(5900) {
			t.Errorf("body mismatch: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"pk-1","orderId":"ord-1","status":"DONE","totalAmount":5900,"method":"CARD"}`))
	})
	p, err := c.ConfirmPayment(context.Background(), "pk-1", "ord-1", 5900)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if p.Status != "DONE" || p.TotalAmount != 5900 || p.Method != "CARD" {
		t.Fatalf("payment mismatch: %+v", p)
	}
}

func TestConfirmPaymentFailureEnvelope(t *testing.T) {
	_, c := stubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"INVALID_AMOUNT","message":"amount mismatch"}`))
	})
	_, err := c.ConfirmPayment(context.Background(), "pk-1", "ord-1", 1)
	var te *TossError
	if !errors.As(err, &te) {
		t.Fatalf("want *TossError, got %T (%v)", err, err)
	}
	if te.Code != "INVALID_AMOUNT" || te.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("envelope mismatch: %+v", te)
	}
}

func TestGetPaymentVerifiesByKey(t *testing.T) {
	_, c := stubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payments/pk-9" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"paymentKey":"pk-9","orderId":"ord-9","status":"CANCELED","totalAmount":12900,"method":"TRANSFER"}`))
	})
	p, err := c.GetPayment(context.Background(), "pk-9")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if p.OrderID != "ord-9" || p.Status != "CANCELED" {
		t.Fatalf("payment mismatch: %+v", p)
	}
}
