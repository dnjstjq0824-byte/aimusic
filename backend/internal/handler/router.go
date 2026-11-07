package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/config"
	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/pkg/storage"
	"ai-music-platform/backend/internal/repository"
	"ai-music-platform/backend/internal/service"
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

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	auth.Use(RateLimit(10, 20))
	auth.POST("/register", authH.Register)
	auth.POST("/register/partner", authH.RegisterPartner)
	auth.POST("/login", authH.Login)
	auth.POST("/refresh", authH.Refresh)
	auth.GET("/me", JWTAuth(d.Services.Auth), authH.Me)
	auth.POST("/withdraw", JWTAuth(d.Services.Auth), authH.Withdraw)
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

	partner := v1.Group("/partner")
	partner.Use(JWTAuth(d.Services.Auth), RequireRole(domain.RolePartner, domain.RoleAdmin))
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
	partner.PUT("/settlement", partnerH.UpdateSettlement)
	partner.POST("/disputes/:id/counter-notice", disputeH.Counter)

	tracks := v1.Group("/tracks")
	tracks.GET("", trackH.ListTracks)
	tracks.GET("/:id", trackH.GetTrack)
	tracks.GET("/:id/stream", JWTAuth(d.Services.Auth), trackH.Stream)

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
	billingAuthed.POST("/orders", billingH.CreateOrder)
	billingAuthed.POST("/payments/confirm", billingH.ConfirmPayment)
	billingAuthed.GET("/subscriptions/me", billingH.MySubscription)
	billingAuthed.POST("/subscriptions/cancel", billingH.CancelSubscription)

	admin := v1.Group("/admin")
	admin.Use(JWTAuth(d.Services.Auth), RequireRole(domain.RoleAdmin))
	admin.GET("/overview", adminH.Overview)
	admin.GET("/pending", adminH.Pending)
	admin.GET("/tracks/pending", adminH.Pending)
	admin.POST("/review", adminH.ReviewTrack)
	admin.POST("/tracks/:id/review", adminH.ReviewTrack)
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
	admin.GET("/partners/:partnerId/settlement", adminH.GetPartnerSettlement)

	return r
}
