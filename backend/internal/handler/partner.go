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
		CollectiveTrustStatus           string   `json:"collectiveTrustStatus"`
		CollectiveName                  *string  `json:"collectiveName"`
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
			CollectiveTrustStatus:           req.Rights.CollectiveTrustStatus,
			CollectiveName:                  req.Rights.CollectiveName,
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

type submitGenerationRequest struct {
	Prompt     string         `json:"prompt" binding:"required"`
	Lyrics     *string        `json:"lyrics"`
	Model      string         `json:"model"`
	LicenseRef string         `json:"licenseRef"`
	Params     map[string]any `json:"params"`
}

// SubmitGeneration enqueues a music-generation job (WI-03 setup-only).
func (h *PartnerHandler) SubmitGeneration(c *gin.Context) {
	var req submitGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	job, err := h.Services.Generation.Submit(c.Request.Context(), service.GenerationSubmitInput{
		PartnerID: partnerID, Prompt: req.Prompt, Lyrics: req.Lyrics,
		Model: req.Model, LicenseRef: req.LicenseRef, Params: req.Params,
	})
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, job)
}

// GetGeneration returns one of the caller's generation jobs.
func (h *PartnerHandler) GetGeneration(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	job, err := h.Services.Generation.Get(c.Request.Context(), partnerID, c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, job)
}

// ListGenerations returns the caller's generation jobs, newest first.
func (h *PartnerHandler) ListGenerations(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	jobs, err := h.Services.Generation.List(c.Request.Context(), partnerID, queryInt(c, "limit", 20))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, jobs)
}

type registerStoreRequest struct {
	BusinessNumber string `json:"businessNumber" binding:"required"`
	StoreName      string `json:"storeName" binding:"required"`
	LicenseCount   int    `json:"licenseCount"`
}

// RegisterStore files a store business for admin verification (WI-05 #19).
func (h *PartnerHandler) RegisterStore(c *gin.Context) {
	var req registerStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	licenses := req.LicenseCount
	if licenses == 0 {
		licenses = 1
	}
	b, err := h.Services.Store.RegisterBusiness(c.Request.Context(), partnerID, req.BusinessNumber, req.StoreName, licenses)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, b)
}

// ListStores returns the caller's store businesses.
func (h *PartnerHandler) ListStores(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	list, err := h.Services.Store.ListBusinesses(c.Request.Context(), partnerID)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, list)
}

type createStoreDeviceRequest struct {
	DeviceName string `json:"deviceName" binding:"required"`
}

// CreateStoreDevice registers a player device and returns its pairing code.
func (h *PartnerHandler) CreateStoreDevice(c *gin.Context) {
	var req createStoreDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	dev, err := h.Services.Store.CreateDevice(c.Request.Context(), partnerID, c.Param("id"), req.DeviceName)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, dev)
}

// ListStoreDevices returns a store's player devices.
func (h *PartnerHandler) ListStoreDevices(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	list, err := h.Services.Store.ListDevices(c.Request.Context(), partnerID, c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, list)
}

type updateStoreDeviceRequest struct {
	DeviceName string `json:"deviceName" binding:"required"`
	Volume     int    `json:"volume"`
}

// UpdateStoreDevice renames or re-volumes a player (remote volume).
func (h *PartnerHandler) UpdateStoreDevice(c *gin.Context) {
	var req updateStoreDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	dev, err := h.Services.Store.UpdateDevice(c.Request.Context(), partnerID, c.Param("deviceId"), req.DeviceName, req.Volume)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, dev)
}

// RevokeStoreDevice deactivates a player.
func (h *PartnerHandler) RevokeStoreDevice(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	if err := h.Services.Store.RevokeDevice(c.Request.Context(), partnerID, c.Param("deviceId")); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"revoked": true})
}

type upsertStoreScheduleRequest struct {
	Weekday    int     `json:"weekday"`
	OpenTime   string  `json:"openTime" binding:"required"`
	CloseTime  string  `json:"closeTime" binding:"required"`
	PlaylistID *string `json:"playlistId"`
	IsClosed   bool    `json:"isClosed"`
}

// UpsertStoreSchedule sets one weekday row for a store.
func (h *PartnerHandler) UpsertStoreSchedule(c *gin.Context) {
	var req upsertStoreScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	s, err := h.Services.Store.UpsertSchedule(c.Request.Context(), partnerID, c.Param("id"), req.Weekday, req.OpenTime, req.CloseTime, req.PlaylistID, req.IsClosed)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, s)
}

// ListStoreSchedules returns a store's weekly schedule.
func (h *PartnerHandler) ListStoreSchedules(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	list, err := h.Services.Store.ListSchedules(c.Request.Context(), partnerID, c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, list)
}

// StoreDeviceLogs returns recent certified plays for a device.
func (h *PartnerHandler) StoreDeviceLogs(c *gin.Context) {
	partnerID, okCall := h.partnerID(c)
	if !okCall {
		return
	}
	logs, err := h.Services.Store.DeviceLogs(c.Request.Context(), partnerID, c.Param("deviceId"), queryInt(c, "limit", 50))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, logs)
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
