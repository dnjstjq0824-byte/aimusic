// Package toss is a minimal Toss Payments API client used for payment
// approval and webhook verification (server-to-server re-query).
package toss

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls the Toss Payments REST API with secret-key basic auth.
type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Toss client. baseURL defaults to the live API host.
func NewClient(secretKey, baseURL string, httpClient *http.Client) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.tosspayments.com"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{secretKey: secretKey, baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

// Payment is the subset of the Toss payment object billing needs.
type Payment struct {
	PaymentKey  string `json:"paymentKey"`
	OrderID     string `json:"orderId"`
	Status      string `json:"status"`
	TotalAmount int    `json:"totalAmount"`
	Method      string `json:"method"`
}

// TossError is a Toss API failure envelope.
type TossError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *TossError) Error() string {
	return fmt.Sprintf("toss %s: %s (http %d)", e.Code, e.Message, e.HTTPStatus)
}

type tossFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.secretKey+":"))
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var f tossFailure
		if err := json.Unmarshal(raw, &f); err != nil || f.Code == "" {
			f = tossFailure{Code: "HTTP_ERROR", Message: strings.TrimSpace(string(raw))}
		}
		return nil, &TossError{HTTPStatus: res.StatusCode, Code: f.Code, Message: f.Message}
	}
	return raw, nil
}

// ConfirmPayment approves a payment (POST /v1/payments/confirm).
func (c *Client) ConfirmPayment(ctx context.Context, paymentKey, orderID string, amount int) (*Payment, error) {
	raw, err := c.do(ctx, http.MethodPost, "/v1/payments/confirm", map[string]any{
		"paymentKey": paymentKey, "orderId": orderID, "amount": amount,
	})
	if err != nil {
		return nil, err
	}
	var p Payment
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPayment re-queries a payment by key (GET /v1/payments/{paymentKey}).
// Webhook handling verifies every event through this call.
func (c *Client) GetPayment(ctx context.Context, paymentKey string) (*Payment, error) {
	raw, err := c.do(ctx, http.MethodGet, "/v1/payments/"+paymentKey, nil)
	if err != nil {
		return nil, err
	}
	var p Payment
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
