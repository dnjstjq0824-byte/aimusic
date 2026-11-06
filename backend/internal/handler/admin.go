package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/repository"
	"ai-music-platform/backend/internal/service"
)

// AdminHandler serves moderation, policy, dispute, curation, and settlement endpoints.
type AdminHandler struct {
	Repos    *repository.Repos
	Services *service.Services
}

// Pending returns the review inbox.
func (h *AdminHandler) Pending(c *gin.Context) {
	queues, err := h.Services.Review.Pending(c.Request.Context())
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, queues)
}

type reviewTrackRequest struct {
	TrackID string `json:"trackId"`
	Approve bool   `json:"approve"`
	Action  string `json:"action"` // optional "APPROVE" | "REJECT"
	Reason  string `json:"reason"`
}

// ReviewTrack approves or rejects a track.
func (h *AdminHandler) ReviewTrack(c *gin.Context) {
	var req reviewTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	trackID := req.TrackID
	if trackID == "" {
		trackID = c.Param("id")
	}
	if trackID == "" {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "trackId required")
		return
	}
	approve := req.Approve
	if req.Action == "APPROVE" {
		approve = true
	} else if req.Action == "REJECT" {
		approve = false
	}
	track, err := h.Services.Review.ReviewTrack(c.Request.Context(), trackID, approve, req.Reason)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, track)
}

type reviewPartnerRequest struct {
	PartnerID string `json:"partnerId" binding:"required"`
	Approve   bool   `json:"approve"`
}

// ReviewPartner approves or suspends a partner.
func (h *AdminHandler) ReviewPartner(c *gin.Context) {
	var req reviewPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	p, err := h.Services.Review.ReviewPartner(c.Request.Context(), req.PartnerID, req.Approve)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, p)
}

type reviewSubscriptionRequest struct {
	SubscriptionID string `json:"subscriptionId" binding:"required"`
	Approve        bool   `json:"approve"`
	ExpiresAt      string `json:"expiresAt"`
}

// ReviewSubscription verifies or rejects a subscription proof.
func (h *AdminHandler) ReviewSubscription(c *gin.Context) {
	var req reviewSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	var expires *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "expiresAt must be RFC3339")
			return
		}
		expires = &t
	}
	sub, err := h.Services.Review.ReviewSubscription(c.Request.Context(), req.SubscriptionID, CurrentUserID(c), req.Approve, expires)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, sub)
}

// ListPolicies returns the provider policy matrix.
func (h *AdminHandler) ListPolicies(c *gin.Context) {
	policies, err := h.Repos.Policies.List(c.Request.Context())
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, policies)
}

type updatePolicyRequest struct {
	AllowsStandaloneStreaming bool     `json:"allowsStandaloneStreaming"`
	AllowedPlanTiers          []string `json:"allowedPlanTiers"`
	BlockedPlanTiers          []string `json:"blockedPlanTiers"`
	SyncOnlyPlanTiers         []string `json:"syncOnlyPlanTiers"`
	MinBitrateKbps            int      `json:"minBitrateKbps" binding:"required"`
	BlockedMetadataKeywords   []string `json:"blockedMetadataKeywords"`
	RejectionMessageKo        string   `json:"rejectionMessageKo" binding:"required"`
	CommercialTermsURL        *string  `json:"commercialTermsUrl"`
}

// UpdatePolicy rewrites a provider policy.
func (h *AdminHandler) UpdatePolicy(c *gin.Context) {
	var req updatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	provider := c.Param("provider")
	if provider == "" {
		provider = c.Param("id")
	}
	p, err := h.Repos.Policies.Update(c.Request.Context(), provider,
		req.AllowsStandaloneStreaming, req.MinBitrateKbps,
		req.BlockedMetadataKeywords, req.RejectionMessageKo, req.CommercialTermsURL,
		req.AllowedPlanTiers, req.BlockedPlanTiers, req.SyncOnlyPlanTiers)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, p)
}

// ListDisputes returns open disputes.
func (h *AdminHandler) ListDisputes(c *gin.Context) {
	disputes, err := h.Repos.Disputes.ListOpen(c.Request.Context(), queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, disputes)
}

type resolveDisputeRequest struct {
	Uphold bool   `json:"uphold"`
	Action string `json:"action"` // optional "TEMP_TAKEDOWN" | "RESTORE" | "PERMANENT_DELETE"
}

// ResolveDispute upholds (takedown + strike) or dismisses a dispute.
func (h *AdminHandler) ResolveDispute(c *gin.Context) {
	var req resolveDisputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	uphold := req.Uphold
	if req.Action == "TEMP_TAKEDOWN" || req.Action == "PERMANENT_DELETE" {
		uphold = true
	} else if req.Action == "RESTORE" {
		uphold = false
	}
	d, err := h.Services.Disputes.Resolve(c.Request.Context(), c.Param("id"), CurrentUserID(c), uphold)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, d)
}

// Curation returns top READY tracks by play count.
func (h *AdminHandler) Curation(c *gin.Context) {
	tracks, err := h.Repos.Tracks.TopByPlays(c.Request.Context(), queryInt(c, "limit", 20))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, tracks)
}

type createCurationPlaylistRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description *string  `json:"description"`
	TrackIDs    []string `json:"trackIds"`
}

// CreateCurationPlaylist creates an official curation playlist and optionally populates tracks.
func (h *AdminHandler) CreateCurationPlaylist(c *gin.Context) {
	var req createCurationPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	p, err := h.Repos.Playlists.CreateFull(c.Request.Context(), CurrentUserID(c), req.Title, req.Description, true, true)
	if err != nil {
		AbortError(c, err)
		return
	}
	for _, tid := range req.TrackIDs {
		_ = h.Services.Playlists.AddTrack(c.Request.Context(), CurrentUserID(c), p.ID, tid)
	}
	created(c, p)
}

type takedownRequest struct {
	TrackID string `json:"trackId" binding:"required"`
	Reason  string `json:"reason"`
}

// Takedown immediately takes down a track.
func (h *AdminHandler) Takedown(c *gin.Context) {
	var req takedownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	track, err := h.Services.Disputes.Takedown(c.Request.Context(), req.TrackID, req.Reason)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, track)
}

type calculateSettlementRequest struct {
	PartnerID      string `json:"partnerId" binding:"required"`
	PeriodStart    string `json:"periodStart" binding:"required"`
	PeriodEnd      string `json:"periodEnd" binding:"required"`
	RatePerPlayKRW int64  `json:"ratePerPlayKrw"`
}

// CalculateSettlement runs settlement calc for a partner period.
func (h *AdminHandler) CalculateSettlement(c *gin.Context) {
	var req calculateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	start, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "periodStart must be YYYY-MM-DD")
		return
	}
	end, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "periodEnd must be YYYY-MM-DD")
		return
	}
	s, err := h.Services.Settlement.Calculate(c.Request.Context(), req.PartnerID, start, end, req.RatePerPlayKRW)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, s)
}

// ListSettlements returns a partner's settlements.
func (h *AdminHandler) ListSettlements(c *gin.Context) {
	settlements, err := h.Repos.Settlements.ListByPartner(c.Request.Context(), c.Param("partnerId"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, settlements)
}

type approveSettlementRequest struct {
	PartnerID   string `json:"partnerId" binding:"required"`
	PeriodStart string `json:"periodStart" binding:"required"`
	PeriodEnd   string `json:"periodEnd" binding:"required"`
}

// GetPartnerSettlement returns bank + masked account preview (never plaintext).
func (h *AdminHandler) GetPartnerSettlement(c *gin.Context) {
	bank, masked, err := h.Services.Settlement.SettlementAccountPreview(c.Request.Context(), c.Param("partnerId"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"bank": bank, "accountMasked": masked})
}

// ApproveSettlement moves a settlement to APPROVED.
func (h *AdminHandler) ApproveSettlement(c *gin.Context) {
	var req approveSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	start, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "periodStart must be YYYY-MM-DD")
		return
	}
	end, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "periodEnd must be YYYY-MM-DD")
		return
	}
	s, err := h.Services.Settlement.Approve(c.Request.Context(), req.PartnerID, start, end)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, s)
}
