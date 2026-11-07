package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// PartnerHandler serves license policies, subscriptions, pre-check, and finalize.
type PartnerHandler struct {
	Repos    *repository.Repos
	Services *service.Services
}

func (h *PartnerHandler) partnerID(c *gin.Context) (string, bool) {
	p, err := h.Repos.Partners.GetByUserID(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return "", false
	}
	return p.ID, true
}

// ListPolicies returns the provider policy matrix.
func (h *PartnerHandler) ListPolicies(c *gin.Context) {
	policies, err := h.Repos.Policies.List(c.Request.Context())
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, policies)
}

// GetPolicy returns one provider policy.
func (h *PartnerHandler) GetPolicy(c *gin.Context) {
	p, err := h.Repos.Policies.Get(c.Request.Context(), c.Param("provider"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, p)
}

// ListSubscriptions returns the caller's subscriptions.
func (h *PartnerHandler) ListSubscriptions(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	subs, err := h.Repos.Subscriptions.ListByPartner(c.Request.Context(), partnerID)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, subs)
}

type createSubscriptionRequest struct {
	Provider       string  `json:"provider" binding:"required"`
	PlanName       string  `json:"planName" binding:"required"`
	ProofObjectKey *string `json:"proofObjectKey"`
}

// CreateSubscription files a subscription proof for admin verification.
func (h *PartnerHandler) CreateSubscription(c *gin.Context) {
	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	sub, err := h.Repos.Subscriptions.Create(c.Request.Context(), partnerID, req.Provider, req.PlanName, req.ProofObjectKey)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, sub)
}

type preCheckRequest struct {
	Provider          string `json:"provider" binding:"required"`
	PlanTierUsed      string `json:"planTierUsed"`
	AudioSHA256       string `json:"audioSha256" binding:"required"`
	FileName          string `json:"fileName" binding:"required"`
	GeneratedAt       string `json:"generatedAt" binding:"required"`
	SubscriptionID    string `json:"subscriptionId"`
	RequiresActiveSub bool   `json:"requiresActiveSub"`
}

// PreCheck runs the license gate and returns a staging upload grant on pass.
func (h *PartnerHandler) PreCheck(c *gin.Context) {
	var req preCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	generatedAt, err := time.Parse(time.RFC3339, req.GeneratedAt)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "generatedAt must be RFC3339")
		return
	}
	out, err := h.Services.Precheck.PreCheck(c.Request.Context(), service.PrecheckInput{
		PartnerID: partnerID, Provider: req.Provider, PlanTierUsed: req.PlanTierUsed,
		AudioSHA256: req.AudioSHA256, FileName: req.FileName, GeneratedAt: generatedAt,
		SubscriptionID: req.SubscriptionID, RequiresActiveSub: req.RequiresActiveSub,
	})
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, out)
}

type finalizeRequest struct {
	StagingKey       string   `json:"stagingKey" binding:"required"`
	Title            string   `json:"title" binding:"required"`
	Provider         string   `json:"provider" binding:"required"`
	PlanTierUsed     string   `json:"planTierUsed"`
	AudioSHA256      string   `json:"audioSha256" binding:"required"`
	GeneratedAt      string   `json:"generatedAt" binding:"required"`
	SubscriptionID   string   `json:"subscriptionId"`
	AlbumID          *string  `json:"albumId"`
	Genre            *string  `json:"genre"`
	MoodTags         []string `json:"moodTags"`
	AIGenerationType string   `json:"aiGenerationType"`
	AIModelVersion   *string  `json:"aiModelVersion"`
	AIPrompt         *string  `json:"aiPrompt"`
	IsPromptPublic   bool     `json:"isPromptPublic"`
	IsInstrumental   bool     `json:"isInstrumental"`
	Lyrics           *string  `json:"lyrics"`
	DurationMs       *int     `json:"durationMs"`
	BitrateKbps      *int     `json:"bitrateKbps"`
	Disclosure       *string  `json:"disclosure"`
	Rights           struct {
		DeclaresOwnership               bool     `json:"declaresOwnership"`
		DeclaresNoVoiceClone            bool     `json:"declaresNoVoiceClone"`
		DeclaresTrainingConsent         bool     `json:"declaresTrainingConsent"`
		ConfirmedNoUnauthorizedSampling bool     `json:"confirmedNoUnauthorizedSampling"`
		ConfirmedIndemnification        bool     `json:"confirmedIndemnification"`
		PlanTierUsed                    *string  `json:"planTierUsed"`
		ProviderTrackURL                *string  `json:"providerTrackUrl"`
		HumanContributionTypes          []string `json:"humanContributionTypes"`
		HumanContributionDesc           *string  `json:"humanContributionDesc"`
		SourceModel                     *string  `json:"sourceModel"`
		SubscriptionID                  *string  `json:"subscriptionId"`
	} `json:"rights"`
}

// Finalize registers an uploaded track and enqueues transcoding.
func (h *PartnerHandler) Finalize(c *gin.Context) {
	var req finalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	generatedAt, err := time.Parse(time.RFC3339, req.GeneratedAt)
	if err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "generatedAt must be RFC3339")
		return
	}
	signedIP := c.ClientIP()
	out, err := h.Services.Finalize.Finalize(c.Request.Context(), service.FinalizeInput{
		PartnerID: partnerID, StagingKey: req.StagingKey, Title: req.Title,
		Provider: req.Provider, PlanTierUsed: req.PlanTierUsed,
		AudioSHA256: req.AudioSHA256, GeneratedAt: generatedAt,
		SubscriptionID: req.SubscriptionID, AlbumID: req.AlbumID,
		Genre: req.Genre, MoodTags: req.MoodTags,
		AIGenerationType: req.AIGenerationType, AIModelVersion: req.AIModelVersion,
		AIPrompt: req.AIPrompt, IsPromptPublic: req.IsPromptPublic,
		IsInstrumental: req.IsInstrumental, Lyrics: req.Lyrics,
		DurationMs: req.DurationMs, BitrateKbps: req.BitrateKbps, Disclosure: req.Disclosure,
		Rights: service.RightsInput{
			DeclaresOwnership:               req.Rights.DeclaresOwnership,
			DeclaresNoVoiceClone:            req.Rights.DeclaresNoVoiceClone,
			DeclaresTrainingConsent:         req.Rights.DeclaresTrainingConsent,
			ConfirmedNoUnauthorizedSampling: req.Rights.ConfirmedNoUnauthorizedSampling,
			ConfirmedIndemnification:        req.Rights.ConfirmedIndemnification,
			PlanTierUsed:                    req.Rights.PlanTierUsed,
			ProviderTrackURL:                req.Rights.ProviderTrackURL,
			HumanContributionTypes:          req.Rights.HumanContributionTypes,
			HumanContributionDesc:           req.Rights.HumanContributionDesc,
			SignedIP:                        &signedIP,
			SourceModel:                     req.Rights.SourceModel,
			SubscriptionID:                  req.Rights.SubscriptionID,
		},
	})
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, out)
}

type createAlbumRequest struct {
	Title          string  `json:"title" binding:"required"`
	ReleaseType    string  `json:"releaseType"`
	CoverObjectKey *string `json:"coverObjectKey"`
}

// CreateAlbum creates an album shell.
func (h *PartnerHandler) CreateAlbum(c *gin.Context) {
	var req createAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	album, err := h.Services.Finalize.CreateAlbum(c.Request.Context(), partnerID, req.Title, req.CoverObjectKey, req.ReleaseType)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, album)
}

// ListAlbums returns the caller's albums.
func (h *PartnerHandler) ListAlbums(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	albums, err := h.Repos.Albums.ListByPartner(c.Request.Context(), partnerID, queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, albums)
}

type updateSettlementRequest struct {
	Bank          string `json:"bank" binding:"required"`
	AccountNumber string `json:"accountNumber" binding:"required"`
}

// UpdateSettlement stores the caller's payout account AES-encrypted.
func (h *PartnerHandler) UpdateSettlement(c *gin.Context) {
	var req updateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	if err := h.Services.Settlement.UpdateSettlementAccount(c.Request.Context(), partnerID, req.Bank, req.AccountNumber); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"updated": true})
}

// ListMyTracks returns the caller's tracks across statuses.
func (h *PartnerHandler) ListMyTracks(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	tracks, err := h.Repos.Tracks.ListByPartner(c.Request.Context(), partnerID, queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, tracks)
}
