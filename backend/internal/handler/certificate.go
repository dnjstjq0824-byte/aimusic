package handler

import (
	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/service"
)

// CertificateHandler serves per-track certificate issuance (admin) and
// public verification by number or by track (P2-6 #24).
type CertificateHandler struct {
	Services *service.Services
}

// IssueCertificate creates a track's certificate (idempotent per track).
func (h *CertificateHandler) IssueCertificate(c *gin.Context) {
	cert, err := h.Services.Certificates.Issue(c.Request.Context(), c.Param("id"), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, cert)
}

// VerifyCertificate resolves a public certificate number.
func (h *CertificateHandler) VerifyCertificate(c *gin.Context) {
	view, err := h.Services.Certificates.Verify(c.Request.Context(), c.Param("number"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, view)
}

// VerifyTrackCertificate resolves a track's certificate (404 when none).
func (h *CertificateHandler) VerifyTrackCertificate(c *gin.Context) {
	view, err := h.Services.Certificates.VerifyByTrack(c.Request.Context(), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, view)
}
