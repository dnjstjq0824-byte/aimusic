package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The abolished per-play rate must fail loudly: a stale caller sending
// ratePerPlayKrw gets 400, never silent new math. Runs before any service
// use, so a zero handler suffices.
func TestCalculateSettlementRejectsLegacyRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"partnerId":"00000000-0000-0000-0000-000000000000",` +
		`"periodStart":"2026-11-01","periodEnd":"2026-12-01","ratePerPlayKrw":10}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/settlements/calculate", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	(&AdminHandler{}).CalculateSettlement(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "abolished") {
		t.Fatalf("body = %s, want abolished-rate message", w.Body.String())
	}
}

func TestCalculateSettlementRejectsBadPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"partnerId":"00000000-0000-0000-0000-000000000000",` +
		`"periodStart":"not-a-date","periodEnd":"2026-12-01","directKrw":1000}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/settlements/calculate", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	(&AdminHandler{}).CalculateSettlement(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
