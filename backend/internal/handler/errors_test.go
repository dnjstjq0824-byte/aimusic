package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
)

func abortRecorder(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	AbortError(c, err)
	return w
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v (%q)", err, w.Body.String())
	}
	return body.Error.Code
}

func TestAbortErrorStatusMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrNotFound, http.StatusNotFound, "ERR_NOT_FOUND"},
		{domain.ErrEmailTaken, http.StatusConflict, "ERR_EMAIL_TAKEN"},
		{domain.ErrInvalidCredentials, http.StatusUnauthorized, "ERR_INVALID_CREDENTIALS"},
		{domain.ErrUserInactive, http.StatusForbidden, "ERR_USER_INACTIVE"},
		{domain.ErrForbidden, http.StatusForbidden, "ERR_FORBIDDEN"},
		{domain.ErrNotStreamable, http.StatusForbidden, "ERR_NOT_STREAMABLE"},
		{domain.ErrPaymentUnavailable, http.StatusServiceUnavailable, "ERR_PAYMENT_UNAVAILABLE"},
		{errors.New("boom"), http.StatusInternalServerError, "ERR_INTERNAL"},
	}
	for _, tc := range cases {
		w := abortRecorder(t, tc.err)
		if w.Code != tc.status {
			t.Errorf("%v: status = %d, want %d", tc.err, w.Code, tc.status)
		}
		if code := errorCode(t, w); code != tc.code {
			t.Errorf("%v: code = %q, want %q", tc.err, code, tc.code)
		}
	}
}

func TestAbortErrorInvalidInputEchoesDetail(t *testing.T) {
	// Handlers wrap with %w; emulate the wrapped shape.
	w := abortRecorder(t, wrapInvalidInput())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if code := errorCode(t, w); code != "ERR_INVALID_INPUT" {
		t.Fatalf("code = %q, want ERR_INVALID_INPUT", code)
	}
}

func wrapInvalidInput() error {
	return &inputError{msg: "email/displayName/password"}
}

type inputError struct{ msg string }

func (e *inputError) Error() string { return "invalid input: " + e.msg }
func (e *inputError) Is(target error) bool {
	return target == domain.ErrInvalidInput
}

func TestAbortErrorGateErrorPassthrough(t *testing.T) {
	gate := &domain.GateError{Code: "ERR_UPSTREAM_TIMEOUT", Message: "upstream timed out"}
	w := abortRecorder(t, gate)
	if w.Code != gate.StatusCode() {
		t.Fatalf("status = %d, want %d", w.Code, gate.StatusCode())
	}
	if code := errorCode(t, w); code != gate.Code {
		t.Fatalf("code = %q, want %q", code, gate.Code)
	}
}

func TestErrCodeForOAuth(t *testing.T) {
	if got := errCodeForOAuth(domain.ErrEmailTaken); got != "ERR_EMAIL_TAKEN" {
		t.Fatalf("got %q", got)
	}
	if got := errCodeForOAuth(errors.New("weird")); got != "ERR_INTERNAL" {
		t.Fatalf("unknown error code = %q, want ERR_INTERNAL", got)
	}
}

func TestQueryInt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newCtx := func(target string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, target, nil)
		return c
	}
	if got := queryInt(newCtx("/x?limit=25"), "limit", 20); got != 25 {
		t.Fatalf("limit = %d, want 25", got)
	}
	if got := queryInt(newCtx("/x"), "limit", 20); got != 20 {
		t.Fatalf("default = %d, want 20", got)
	}
	if got := queryInt(newCtx("/x?limit=abc"), "limit", 20); got != 20 {
		t.Fatalf("garbage = %d, want default 20", got)
	}
}
