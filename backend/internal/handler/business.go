package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/service"
)

// BusinessHandler serves public B2B inquiries, the waitlist, and admin
// inquiry management (R7).
type BusinessHandler struct {
	Business *service.BusinessService
}

type inquiryRequest struct {
	Company      string `json:"company"`
	Country      string `json:"country"`
	StoreCount   int    `json:"storeCount"`
	UseCase      string `json:"useCase"`
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	Message      string `json:"message"`
	// Honeypot: real clients leave it empty; bots fill it.
	Website string `json:"website"`
}

// SubmitInquiry stores a B2B inquiry. Honeypot fills get a fake success.
func (h *BusinessHandler) SubmitInquiry(c *gin.Context) {
	var req inquiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: invalid inquiry body", domain.ErrInvalidInput))
		return
	}
	if strings.TrimSpace(req.Website) != "" {
		ok(c, gin.H{"received": true})
		return
	}
	q, err := h.Business.SubmitInquiry(c.Request.Context(), service.InquiryInput{
		Company: req.Company, Country: req.Country, StoreCount: req.StoreCount,
		UseCase: req.UseCase, ContactName: req.ContactName, ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone, Message: req.Message,
	})
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, gin.H{"id": q.ID, "received": true})
}

// ListInquiries returns newest-first inquiries (admin).
func (h *BusinessHandler) ListInquiries(c *gin.Context) {
	items, err := h.Business.ListInquiries(c.Request.Context(),
		strings.ToUpper(strings.TrimSpace(c.Query("status"))), queryInt(c, "limit", 50))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, items)
}

type inquiryStatusRequest struct {
	Status string `json:"status"`
}

// SetInquiryStatus moves one inquiry (admin).
func (h *BusinessHandler) SetInquiryStatus(c *gin.Context) {
	var req inquiryStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: status required", domain.ErrInvalidInput))
		return
	}
	q, err := h.Business.SetInquiryStatus(c.Request.Context(), c.Param("id"),
		strings.ToUpper(strings.TrimSpace(req.Status)))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, q)
}

type waitlistRequest struct {
	Email    string `json:"email"`
	Interest string `json:"interest"`
	Locale   string `json:"locale"`
}

// JoinWaitlist registers an email (idempotent).
func (h *BusinessHandler) JoinWaitlist(c *gin.Context) {
	var req waitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: email required", domain.ErrInvalidInput))
		return
	}
	w, existed, err := h.Business.JoinWaitlist(c.Request.Context(), req.Email, req.Interest, req.Locale)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"id": w.ID, "registered": !existed})
}
