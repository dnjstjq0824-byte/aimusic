package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/repository"
)

// LegalHandler serves legal terms and consent records.
type LegalHandler struct {
	Repos *repository.Repos
}

// ListTerms returns all legal terms.
func (h *LegalHandler) ListTerms(c *gin.Context) {
	terms, err := h.Repos.Legal.ListTerms(c.Request.Context())
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, terms)
}

// GetTerm returns the latest term for a code.
func (h *LegalHandler) GetTerm(c *gin.Context) {
	term, err := h.Repos.Legal.GetTermByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, term)
}

type consentRequest struct {
	TermsID string `json:"termsId" binding:"required"`
}

// CreateConsent records the caller's consent to a term.
func (h *LegalHandler) CreateConsent(c *gin.Context) {
	var req consentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	if err := h.Repos.Legal.CreateConsent(c.Request.Context(), CurrentUserID(c), req.TermsID, &ip, &ua); err != nil {
		AbortError(c, err)
		return
	}
	created(c, gin.H{"consented": true})
}

// ListMyConsents returns the caller's consent records.
func (h *LegalHandler) ListMyConsents(c *gin.Context) {
	consents, err := h.Repos.Legal.ListConsentsByUser(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, consents)
}
