package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// AdminHandler serves moderation, policy, dispute, curation, and settlement endpoints.
type AdminHandler struct {
	Repos    *repository.Repos
	Services *service.Services
}

// recordAudit appends to the admin action trail. Audit writes never fail the
// admin action itself; failures are logged for operator follow-up.
func (h *AdminHandler) recordAudit(c *gin.Context, action, entityType, entityID string, metadata map[string]any) {
	actor := CurrentUserID(c)
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}
	meta := "{}"
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			meta = string(b)
		}
	}
	if err := h.Repos.Audit.Record(c.Request.Context(), actorPtr, action, entityType, entityID, meta, c.ClientIP()); err != nil {
		slog.Error("audit record failed", "action", action, "entity", entityType+"/"+entityID, "err", err)
	}
}

// ListAuditEvents returns the admin action trail, newest first.
func (h *AdminHandler) ListAuditEvents(c *gin.Context) {
	events, err := h.Repos.Audit.List(c.Request.Context(),
		c.Query("action"), c.Query("entityType"),
		queryInt(c, "limit", 50), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, events)
}

// ListUserBlocks returns one account's strike/block history for admin review.
func (h *AdminHandler) ListUserBlocks(c *gin.Context) {
	blocks, err := h.Repos.Blocks.ListByUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, blocks)
}

// ListUsers searches accounts for admin user management.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.Repos.Users.Search(c.Request.Context(),
		c.Query("q"), c.Query("role"),
		queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, users)
}

type setUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// SetUserRole changes an account's role. Self-demotion and demoting the last
// admin are refused so the console cannot lock itself out.
func (h *AdminHandler) SetUserRole(c *gin.Context) {
	var req setUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	id := c.Param("id")
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role != domain.RoleAdmin && role != domain.RolePartner && role != domain.RoleUser {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "role must be ADMIN|PARTNER|USER")
		return
	}
	if id == CurrentUserID(c) {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "cannot change your own role")
		return
	}
	target, err := h.Repos.Users.GetByID(c.Request.Context(), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	if target.Role == domain.RoleAdmin && role != domain.RoleAdmin {
		n, err := h.Repos.Users.CountByRole(c.Request.Context(), domain.RoleAdmin)
		if err != nil {
			AbortError(c, err)
			return
		}
		if n <= 1 {
			abort(c, http.StatusConflict, "ERR_INVALID_INPUT", "cannot demote the last admin")
			return
		}
	}
	if err := h.Repos.Users.SetRole(c.Request.Context(), id, role); err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "USER_SET_ROLE", "user", id, map[string]any{"role": role, "via": "console"})
	updated, err := h.Repos.Users.GetByID(c.Request.Context(), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, updated)
}

type setUserActiveRequest struct {
	Active bool `json:"active"`
}

// SetUserActive locks or unlocks an account (self-lock refused).
func (h *AdminHandler) SetUserActive(c *gin.Context) {
	var req setUserActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	id := c.Param("id")
	if id == CurrentUserID(c) && !req.Active {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "cannot lock your own account")
		return
	}
	if err := h.Repos.Users.SetActive(c.Request.Context(), id, req.Active); err != nil {
		AbortError(c, err)
		return
	}
	if !req.Active {
		_ = h.Repos.Users.BumpTokenVersion(c.Request.Context(), id)
	}
	h.recordAudit(c, "USER_SET_ACTIVE", "user", id, map[string]any{"active": req.Active, "via": "console"})
	updated, err := h.Repos.Users.GetByID(c.Request.Context(), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, updated)
}

// Overview returns the admin console landing snapshot.
func (h *AdminHandler) Overview(c *gin.Context) {
	overview, err := h.Services.Overview.Overview(c.Request.Context())
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, overview)
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
	h.recordAudit(c, "TRACK_REVIEW", "track", trackID, map[string]any{"approve": approve, "reason": req.Reason})
	ok(c, track)
}

type setISRCRequest struct {
	ISRC string `json:"isrc" binding:"required"`
}

// SetTrackISRC assigns a distribution ISRC to a track (WI-05).
func (h *AdminHandler) SetTrackISRC(c *gin.Context) {
	var req setISRCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	trackID := c.Param("id")
	if trackID == "" {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "track id required")
		return
	}
	track, err := h.Services.Review.SetTrackISRC(c.Request.Context(), trackID, req.ISRC)
	if err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "TRACK_ISRC_ASSIGN", "track", trackID, map[string]any{"isrc": track.ISRC})
	ok(c, track)
}

// DSPKit exports the distributor handoff kit for selected tracks (WI-05 #32).
// GET /admin/distribution/dsp-kit?trackIds=a,b&format=json|csv
func (h *AdminHandler) DSPKit(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("trackIds"))
	if raw == "" {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "trackIds required (comma-separated, max 100)")
		return
	}
	kit, err := h.Services.Distribution.BuildDSPKit(c.Request.Context(), strings.Split(raw, ","))
	if err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "DSP_KIT_EXPORT", "distribution", raw, map[string]any{"tracks": len(kit.Tracks), "skipped": len(kit.Skipped)})
	if strings.ToLower(strings.TrimSpace(c.Query("format"))) == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="dsp-kit-`+kit.GeneratedAt+`.csv"`)
		c.String(http.StatusOK, "%s", service.BuildDSPKitCSV(kit))
		return
	}
	ok(c, kit)
}

type verifyStoreRequest struct {
	Approve bool `json:"approve"`
}

// VerifyStore approves or suspends a store business (WI-05 #19).
func (h *AdminHandler) VerifyStore(c *gin.Context) {
	var req verifyStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	b, err := h.Services.Store.VerifyBusiness(c.Request.Context(), c.Param("id"), CurrentUserID(c), req.Approve)
	if err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "STORE_VERIFY", "store_business", c.Param("id"), map[string]any{"approve": req.Approve})
	ok(c, b)
}

// ListStores returns store businesses filtered by status.
func (h *AdminHandler) ListStores(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if status == "" {
		status = domain.StorePending
	}
	switch status {
	case domain.StorePending, domain.StoreApproved, domain.StoreSuspended:
	default:
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "status must be PENDING|APPROVED|SUSPENDED")
		return
	}
	list, err := h.Repos.Store.ListBusinessesByStatus(c.Request.Context(), status, queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, list)
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
	h.recordAudit(c, "PARTNER_REVIEW", "partner", req.PartnerID, map[string]any{"approve": req.Approve})
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
	h.recordAudit(c, "SUBSCRIPTION_REVIEW", "subscription", req.SubscriptionID, map[string]any{"approve": req.Approve})
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
	MaxGeneratedAt            *string  `json:"maxGeneratedAt"`
	TermsSnapshotURL          *string  `json:"termsSnapshotUrl"`
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
	var maxGen *time.Time
	if req.MaxGeneratedAt != nil && *req.MaxGeneratedAt != "" {
		t, err := time.Parse(time.RFC3339, *req.MaxGeneratedAt)
		if err != nil {
			abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "maxGeneratedAt must be RFC3339")
			return
		}
		maxGen = &t
	}
	p, err := h.Repos.Policies.Update(c.Request.Context(), provider,
		req.AllowsStandaloneStreaming, req.MinBitrateKbps,
		req.BlockedMetadataKeywords, req.RejectionMessageKo, req.CommercialTermsURL,
		req.AllowedPlanTiers, req.BlockedPlanTiers, req.SyncOnlyPlanTiers,
		maxGen, req.TermsSnapshotURL)
	if err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "POLICY_UPDATE", "policy", provider, nil)
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
	h.recordAudit(c, "DISPUTE_RESOLVE", "dispute", c.Param("id"), map[string]any{"uphold": uphold, "action": req.Action})
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
	h.recordAudit(c, "CURATION_CREATE", "playlist", p.ID, map[string]any{"title": req.Title, "trackCount": len(req.TrackIDs)})
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
	h.recordAudit(c, "TRACK_TAKEDOWN", "track", req.TrackID, map[string]any{"reason": req.Reason})
	ok(c, track)
}

type calculateSettlementRequest struct {
	PartnerID      string `json:"partnerId" binding:"required"`
	PeriodStart    string `json:"periodStart" binding:"required"`
	PeriodEnd      string `json:"periodEnd" binding:"required"`
	RatePerPlayKRW int64  `json:"ratePerPlayKrw"`
	IncomeType     string `json:"incomeType"`
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
	s, err := h.Services.Settlement.Calculate(c.Request.Context(), req.PartnerID, start, end, req.RatePerPlayKRW, req.IncomeType)
	if err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "SETTLEMENT_CALCULATE", "settlement", req.PartnerID, map[string]any{"periodStart": req.PeriodStart, "periodEnd": req.PeriodEnd})
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
	h.recordAudit(c, "SETTLEMENT_APPROVE", "settlement", req.PartnerID, map[string]any{"periodStart": req.PeriodStart, "periodEnd": req.PeriodEnd})
	ok(c, s)
}

type markPaidRequest struct {
	PartnerID   string `json:"partnerId" binding:"required"`
	PeriodStart string `json:"periodStart" binding:"required"`
	PeriodEnd   string `json:"periodEnd" binding:"required"`
	Reference   string `json:"reference"`
}

// MarkPaidSettlement records an executed bank transfer (APPROVED -> PAID).
func (h *AdminHandler) MarkPaidSettlement(c *gin.Context) {
	var req markPaidRequest
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
	s, err := h.Services.Settlement.MarkPaid(c.Request.Context(), req.PartnerID, start, end, req.Reference)
	if err != nil {
		AbortError(c, err)
		return
	}
	h.recordAudit(c, "SETTLEMENT_MARK_PAID", "settlement", req.PartnerID, map[string]any{"periodStart": req.PeriodStart, "periodEnd": req.PeriodEnd, "reference": req.Reference})
	ok(c, s)
}
