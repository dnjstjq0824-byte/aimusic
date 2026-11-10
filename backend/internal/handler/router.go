package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// Deps wires handlers to services and infrastructure.
type Deps struct {
	Config   *config.Config
	Repos    *repository.Repos
	Services *service.Services
	Storage  storage.Storage
	Buckets  storage.Buckets
}

// SetupRouter registers health, public, partner, and admin routes.
func SetupRouter(d *Deps) *gin.Engine {
	r := gin.New()
	// Trust X-Forwarded-For only from configured proxies so ClientIP (used by
	// rate limiting and audit logs) cannot be spoofed. Empty trusts none.
	proxies := d.Config.Security.TrustedProxies
	if proxies == nil {
		proxies = []string{}
	}
	_ = r.SetTrustedProxies(proxies)
	r.Use(gin.Recovery(), RequestID(), SecurityHeaders(d.Config), CORS(d.Config), AuditLog(), RateLimit(100, 200))

	// Liveness: the process answers. Readiness (/readyz) also gates on deps.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
	})
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		checks := gin.H{}
		ready := true
		if err := d.Repos.DB.PingContext(ctx); err != nil {
			ready = false
			checks["db"] = "down"
		} else {
			checks["db"] = "up"
		}
		if d.Repos.RDB == nil || d.Repos.RDB.Ping(ctx).Err() != nil {
			ready = false
			checks["redis"] = "down"
		} else {
			checks["redis"] = "up"
		}
		if depth, err := d.Repos.Queue.QueueDepth(ctx); err == nil {
			checks["queueDepth"] = depth
		}
		if n, err := d.Repos.Jobs.CountFailedSince(ctx, time.Now().Add(-24*time.Hour)); err == nil {
			checks["jobsFailed24h"] = n
		}
		if ready {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "checks": checks})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "checks": checks})
		}
	})

	authH := &AuthHandler{Auth: d.Services.Auth, Resets: d.Services.PasswordReset, OAuth: d.Services.OAuth, FrontendBase: d.Config.OAuth.FrontendBaseURL}
	legalH := &LegalHandler{Repos: d.Repos}
	partnerH := &PartnerHandler{Repos: d.Repos, Services: d.Services}
	trackH := &TrackHandler{Repos: d.Repos, Config: d.Config, Storage: d.Storage, Buckets: d.Buckets}
	playbackH := &PlaybackHandler{Playback: d.Services.Playback}
	playlistH := &PlaylistHandler{Playlists: d.Services.Playlists}
	disputeH := &DisputeHandler{Repos: d.Repos, Disputes: d.Services.Disputes}
	adminH := &AdminHandler{Repos: d.Repos, Services: d.Services}
	billingH := &BillingHandler{Billing: d.Services.Billing}
	deviceH := &DeviceHandler{Repos: d.Repos, Services: d.Services, Config: d.Config, Storage: d.Storage, Buckets: d.Buckets}
	certH := &CertificateHandler{Services: d.Services}

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	auth.Use(RateLimit(10, 20))
	auth.POST("/register", authH.Register)
	auth.POST("/register/partner", authH.RegisterPartner)
	auth.POST("/login", authH.Login)
	auth.POST("/refresh", authH.Refresh)
	auth.GET("/me", JWTAuth(d.Services.Auth), authH.Me)
	auth.POST("/age-confirm", JWTAuth(d.Services.Auth), authH.ConfirmAge)
	auth.GET("/export", JWTAuth(d.Services.Auth), authH.ExportData)
	auth.POST("/withdraw", JWTAuth(d.Services.Auth), authH.Withdraw)
	auth.POST("/logout-all", JWTAuth(d.Services.Auth), authH.LogoutAll)
	auth.POST("/totp/enroll", JWTAuth(d.Services.Auth), authH.EnrollTOTP)
	auth.POST("/totp/confirm", JWTAuth(d.Services.Auth), authH.ConfirmTOTPEnrollment)
	auth.POST("/totp/disable", JWTAuth(d.Services.Auth), authH.DisableTOTP)
	auth.POST("/password-reset/request", authH.RequestPasswordReset)
	auth.POST("/password-reset/confirm", authH.ConfirmPasswordReset)
	auth.GET("/oauth/providers", authH.OAuthProviders)
	auth.GET("/oauth/:provider/start", authH.OAuthStart)
	auth.GET("/oauth/:provider/callback", authH.OAuthCallback)

	legal := v1.Group("/legal")
	legal.GET("/terms", legalH.ListTerms)
	legal.GET("/terms/:code", legalH.GetTerm)
	legal.POST("/consents", JWTAuth(d.Services.Auth), legalH.CreateConsent)
	legal.GET("/consents/me", JWTAuth(d.Services.Auth), legalH.ListMyConsents)
	legal.GET("/consents/pending", JWTAuth(d.Services.Auth), legalH.ListPendingConsents)

	partner := v1.Group("/partner")
	partner.Use(JWTAuth(d.Services.Auth), RequireRole(domain.RolePartner, domain.RoleAdmin), RequireAgeConfirmed(d.Repos.Users), RequireConsents(d.Repos.Legal))
	partner.GET("/license-policies", partnerH.ListPolicies)
	partner.GET("/license-policies/:provider", partnerH.GetPolicy)
	partner.GET("/subscriptions", partnerH.ListSubscriptions)
	partner.POST("/subscriptions", partnerH.CreateSubscription)
	partner.POST("/pre-check", partnerH.PreCheck)
	partner.POST("/tracks/pre-check", partnerH.PreCheck)
	partner.POST("/finalize", partnerH.Finalize)
	partner.POST("/tracks/finalize", partnerH.Finalize)
	partner.POST("/albums", partnerH.CreateAlbum)
	partner.GET("/albums", partnerH.ListAlbums)
	partner.GET("/tracks", partnerH.ListMyTracks)
	partner.POST("/generations", partnerH.SubmitGeneration)
	partner.GET("/generations", partnerH.ListGenerations)
	partner.GET("/generations/:id", partnerH.GetGeneration)
	partner.POST("/stores", partnerH.RegisterStore)
	partner.GET("/stores", partnerH.ListStores)
	partner.POST("/stores/:id/devices", partnerH.CreateStoreDevice)
	partner.GET("/stores/:id/devices", partnerH.ListStoreDevices)
	partner.PUT("/stores/:id/devices/:deviceId", partnerH.UpdateStoreDevice)
	partner.POST("/stores/:id/devices/:deviceId/revoke", partnerH.RevokeStoreDevice)
	partner.GET("/stores/:id/schedules", partnerH.ListStoreSchedules)
	partner.POST("/stores/:id/schedules", partnerH.UpsertStoreSchedule)
	partner.GET("/stores/:id/devices/:deviceId/logs", partnerH.StoreDeviceLogs)
	partner.PUT("/settlement", partnerH.UpdateSettlement)
	partner.POST("/disputes/:id/counter-notice", disputeH.Counter)

	tracks := v1.Group("/tracks")
	tracks.GET("", trackH.ListTracks)
	tracks.GET("/:id", trackH.GetTrack)
	tracks.GET("/:id/certificate", certH.VerifyTrackCertificate)
	certs := v1.Group("/certificates")
	certs.GET("/:number", certH.VerifyCertificate)
	tracks.GET("/:id/stream", JWTAuth(d.Services.Auth), RequireAgeConfirmed(d.Repos.Users), RequireConsents(d.Repos.Legal), trackH.Stream)
	tracks.GET("/:id/download", JWTAuth(d.Services.Auth), RequireAgeConfirmed(d.Repos.Users), RequireConsents(d.Repos.Legal), trackH.Download)

	playback := v1.Group("/playback")
	playback.Use(JWTAuth(d.Services.Auth))
	playback.POST("/heartbeat", playbackH.Heartbeat)
	playback.GET("/state/:trackId", playbackH.GetState)
	playback.PUT("/state/:trackId", playbackH.SetState)
	playback.PUT("/state", playbackH.SetState)

	playlists := v1.Group("/playlists")
	playlists.Use(JWTAuth(d.Services.Auth))
	playlists.GET("", playlistH.ListMine)
	playlists.GET("/me", playlistH.ListMine)
	playlists.POST("", playlistH.Create)
	playlists.GET("/:id", playlistH.Get)
	playlists.PUT("/:id", playlistH.Update)
	playlists.DELETE("/:id", playlistH.Delete)
	playlists.GET("/:id/tracks", playlistH.ListTracks)
	playlists.POST("/:id/tracks", playlistH.AddTrack)
	playlists.DELETE("/:id/tracks/:trackId", playlistH.RemoveTrack)
	playlists.PUT("/:id/reorder", playlistH.Reorder)
	playlists.PUT("/:id/tracks/reorder", playlistH.Reorder)

	disputes := v1.Group("/disputes")
	disputes.Use(JWTAuth(d.Services.Auth))
	disputes.POST("/report", disputeH.Report)
	disputes.POST("/:id/counter", disputeH.Counter)
	disputes.GET("/blocks", disputeH.MyBlocks)

	users := v1.Group("/users")
	users.Use(JWTAuth(d.Services.Auth))
	users.POST("/blocks", disputeH.BlockArtist)
	users.GET("/blocks", disputeH.ListArtistBlocks)
	users.DELETE("/blocks/:partnerId", disputeH.UnblockArtist)

	billing := v1.Group("/billing")
	billing.GET("/plans", billingH.ListPlans)
	billing.POST("/webhooks/toss", billingH.TossWebhook)
	billingAuthed := billing.Group("")
	billingAuthed.Use(JWTAuth(d.Services.Auth))
	billingAuthed.POST("/orders", RequireAgeConfirmed(d.Repos.Users), RequireConsents(d.Repos.Legal), billingH.CreateOrder)
	billingAuthed.POST("/payments/confirm", RequireAgeConfirmed(d.Repos.Users), RequireConsents(d.Repos.Legal), billingH.ConfirmPayment)
	billingAuthed.GET("/subscriptions/me", billingH.MySubscription)
	billingAuthed.POST("/subscriptions/cancel", billingH.CancelSubscription)
	billingAuthed.POST("/payments/refund", billingH.Refund)

	admin := v1.Group("/admin")
	admin.Use(JWTAuth(d.Services.Auth), RequireRole(domain.RoleAdmin))
	admin.GET("/overview", adminH.Overview)
	admin.GET("/pending", adminH.Pending)
	admin.GET("/audit-events", adminH.ListAuditEvents)
	admin.GET("/users", adminH.ListUsers)
	admin.POST("/users/:id/role", adminH.SetUserRole)
	admin.POST("/users/:id/active", adminH.SetUserActive)
	admin.GET("/users/:id/blocks", adminH.ListUserBlocks)
	admin.GET("/tracks/pending", adminH.Pending)
	admin.POST("/review", adminH.ReviewTrack)
	admin.POST("/tracks/:id/review", adminH.ReviewTrack)
	admin.PUT("/tracks/:id/isrc", adminH.SetTrackISRC)
	admin.POST("/tracks/:id/certificate", certH.IssueCertificate)
	admin.GET("/distribution/dsp-kit", adminH.DSPKit)
	admin.POST("/review/partner", adminH.ReviewPartner)
	admin.POST("/review/subscription", adminH.ReviewSubscription)
	admin.GET("/policies", adminH.ListPolicies)
	admin.PUT("/policies/:provider", adminH.UpdatePolicy)
	admin.PUT("/license-policies/:id", adminH.UpdatePolicy)
	admin.GET("/disputes", adminH.ListDisputes)
	admin.POST("/disputes/:id/resolve", adminH.ResolveDispute)
	admin.POST("/disputes/:id/action", adminH.ResolveDispute)
	admin.GET("/curation", adminH.Curation)
	admin.POST("/curation/playlists", adminH.CreateCurationPlaylist)
	admin.POST("/curation/takedown", adminH.Takedown)
	admin.POST("/settlements/calculate", adminH.CalculateSettlement)
	admin.GET("/settlements/:partnerId", adminH.ListSettlements)
	admin.POST("/settlements/approve", adminH.ApproveSettlement)
	admin.POST("/settlements/mark-paid", adminH.MarkPaidSettlement)
	admin.POST("/settlements/hold", adminH.HoldSettlement)
	admin.POST("/settlements/release", adminH.ReleaseSettlement)
	admin.GET("/partners/:partnerId/settlement", adminH.GetPartnerSettlement)
	admin.GET("/stores", adminH.ListStores)
	admin.POST("/stores/:id/verify", adminH.VerifyStore)

	storeDevice := v1.Group("/store/device")
	storeDevice.POST("/claim", deviceH.ClaimDevice)
	storeDevice.Use(StoreDeviceAuth(d.Services.Store))
	storeDevice.GET("/me", deviceH.DeviceMe)
	storeDevice.GET("/queue", deviceH.DeviceQueue)
	storeDevice.POST("/heartbeat", deviceH.DeviceHeartbeat)
	storeDevice.GET("/stream/:id", deviceH.DeviceStream)

	return r
}
