package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/service"
)

// BillingHandler serves plan catalog, Toss orders, approval, webhooks,
// and the caller's subscription.
type BillingHandler struct {
	Billing *service.BillingService
}

// ListPlans returns the purchasable plan catalog.
func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.Billing.Plans(c.Request.Context())
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, plans)
}

type createOrderRequest struct {
	PlanCode        string `json:"planCode" binding:"required"`
	RefundConsentAt string `json:"refundConsentAt" binding:"required"`
}

// CreateOrder opens a PENDING merchant order for the Toss checkout.
func (h *BillingHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	consentAt, err := time.Parse(time.RFC3339, req.RefundConsentAt)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "refundConsentAt must be RFC3339")
		return
	}
	order, plan, err := h.Billing.CreateOrder(c.Request.Context(), CurrentUserID(c), req.PlanCode, consentAt)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, gin.H{"order": order, "plan": plan})
}

type confirmPaymentRequest struct {
	PaymentKey string `json:"paymentKey" binding:"required"`
	OrderID    string `json:"orderId" binding:"required"`
	Amount     int    `json:"amount" binding:"required"`
}

// ConfirmPayment approves a Toss payment and activates the subscription.
func (h *BillingHandler) ConfirmPayment(c *gin.Context) {
	var req confirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	sub, err := h.Billing.Confirm(c.Request.Context(), CurrentUserID(c), req.PaymentKey, req.OrderID, req.Amount)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"subscription": sub})
}

// TossWebhook applies Toss transmissions after server-to-server verification.
// Failures return 4xx/5xx so Toss retries; duplicates return 200.
func (h *BillingHandler) TossWebhook(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil || len(raw) == 0 {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "empty webhook body")
		return
	}
	if err := h.Billing.HandleWebhook(c.Request.Context(), raw); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"received": true})
}

// MySubscription returns the caller's ACTIVE subscription (null when none).
func (h *BillingHandler) MySubscription(c *gin.Context) {
	sub, err := h.Billing.MySubscription(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"subscription": sub})
}

// CancelSubscription ends the caller's ACTIVE pass.
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	if err := h.Billing.CancelSubscription(c.Request.Context(), CurrentUserID(c)); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"cancelled": true})
}

type refundRequest struct {
	// OrderID is the merchant order ID; empty refunds the ACTIVE pass.
	OrderID string `json:"orderId"`
	Reason  string `json:"reason"`
}

// Refund voids a PAID order at the PG and ends the pass immediately.
func (h *BillingHandler) Refund(c *gin.Context) {
	var req refundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	order, err := h.Billing.Refund(c.Request.Context(), CurrentUserID(c), req.OrderID, req.Reason)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"order": order})
}
