package domain

import "time"

// Roles (users.role).
const (
	RoleAdmin   = "ADMIN"
	RolePartner = "PARTNER"
	RoleUser    = "USER"
)

// AI providers shared by policies, subscriptions, and tracks.
const (
	ProviderSuno       = "SUNO"
	ProviderUdio       = "UDIO"
	ProviderAiva       = "AIVA"
	ProviderStable     = "STABLE_AUDIO"
	ProviderSoundraw   = "SOUNDRAW"
	ProviderMubert     = "MUBERT"
	ProviderElevenLabs = "ELEVENLABS"
	ProviderCustomOpen = "CUSTOM_OPEN_MODEL"
)

// ValidProvider reports whether p is a known AI provider.
func ValidProvider(p string) bool {
	switch p {
	case ProviderSuno, ProviderUdio, ProviderAiva, ProviderStable,
		ProviderSoundraw, ProviderMubert, ProviderElevenLabs, ProviderCustomOpen:
		return true
	}
	return false
}

// Partner profile statuses.
const (
	PartnerPending    = "PENDING"
	PartnerApproved   = "APPROVED"
	PartnerSuspended  = "SUSPENDED"
	PartnerTerminated = "TERMINATED"
)

// Subscription statuses (partner_ai_subscriptions.status).
const (
	SubPending  = "PENDING"
	SubVerified = "VERIFIED"
	SubRejected = "REJECTED"
	SubExpired  = "EXPIRED"
)

// Legal term codes.
const (
	TermTOS             = "TOS"
	TermPartnerContract = "PARTNER_CONTRACT"
	TermPrivacy         = "PRIVACY"
	TermCopyright       = "COPYRIGHT_POLICY"
)

// Album statuses and release types.
const (
	AlbumDraft         = "DRAFT"
	AlbumPendingReview = "PENDING_REVIEW"
	AlbumLive          = "LIVE"
	AlbumTakenDown     = "TAKEN_DOWN"
)

// Track statuses.
const (
	TrackUploaded            = "UPLOADED"
	TrackLicenseGatePending  = "LICENSE_GATE_PENDING"
	TrackLicenseGateRejected = "LICENSE_GATE_REJECTED"
	TrackTranscoding         = "TRANSCODING"
	TrackReady               = "READY"
	TrackTakenDown           = "TAKEN_DOWN"
)

// Transcoding job statuses.
const (
	JobQueued    = "QUEUED"
	JobRunning   = "RUNNING"
	JobSucceeded = "SUCCEEDED"
	JobFailed    = "FAILED"
)

// Dispute reasons and statuses.
const (
	DisputeAIPolicy  = "AI_POLICY_VIOLATION"
	DisputeVoice     = "VOICE_CLONE"
	DisputeSample    = "UNAUTHORIZED_SAMPLE"
	DisputeOther     = "OTHER"
	DisputeOpen      = "OPEN"
	DisputeReview    = "UNDER_REVIEW"
	DisputeUpheld    = "UPHELD_TAKEDOWN"
	DisputeDismissed = "DISMISSED"
)

// Settlement statuses.
const (
	SettlePending   = "PENDING"
	SettleApproved  = "APPROVED"
	SettlePaid      = "PAID"
	SettleForfeited = "FORFEITED"
	// SettleHeld suspends a PENDING settlement while confirmed fraud is
	// reviewed (P2-2 #12). Release returns it to PENDING.
	SettleHeld = "HELD"
)

// Settlement withholding income types (partner_settlements.income_type).
const (
	// IncomeBusiness33 is continuing business income: 3.3% withholding (33/1000).
	IncomeBusiness33 = "BUSINESS_33"
	// IncomeOther88 is occasional other income: 8.8% withholding (88/1000).
	IncomeOther88 = "OTHER_88"
)

// ValidIncomeType reports whether t is a known withholding income type.
func ValidIncomeType(t string) bool {
	return t == IncomeBusiness33 || t == IncomeOther88
}

// Collective-trust declaration values (track_rights_declarations.collective_trust_status).
const (
	// CollectiveNone means no collecting-society trust (default).
	CollectiveNone = "NONE"
	// CollectiveTrusted means the track is trusted to a society (e.g. KOMCA);
	// trusted tracks are refused until a society settlement path exists.
	CollectiveTrusted = "TRUSTED"
)

// ValidDisputeReason reports whether r is a known dispute reason.
func ValidDisputeReason(r string) bool {
	switch r {
	case DisputeAIPolicy, DisputeVoice, DisputeSample, DisputeOther,
		"NON_COMMERCIAL_VIOLATION", "COPYRIGHT", "VOICE_PUBLICITY", "TRADEMARK", "ILLEGAL_CONTENT":
		return true
	}
	return false
}

// NormalizeDisputeReason maps spec alias codes to canonical DB check values.
func NormalizeDisputeReason(r string) string {
	switch r {
	case "NON_COMMERCIAL_VIOLATION":
		return DisputeAIPolicy
	case "VOICE_PUBLICITY":
		return DisputeVoice
	case "COPYRIGHT":
		return DisputeSample
	case "TRADEMARK", "ILLEGAL_CONTENT":
		return DisputeOther
	default:
		return r
	}
}

// User mirrors the users table.
type User struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"`
	Role           string     `json:"role"`
	DisplayName    string     `json:"displayName"`
	IsActive       bool       `json:"isActive"`
	BirthYear      *int       `json:"birthYear,omitempty"`
	AgeConfirmedAt *time.Time `json:"ageConfirmedAt,omitempty"`
	TokenVersion   int        `json:"-"`
	TOTPEnabled    bool       `json:"totpEnabled"`
	TOTPEnrolledAt *time.Time `json:"totpEnrolledAt,omitempty"`
	TOTPSecret     string     `json:"-"`
	// TOTPRecoveryLeft counts unused recovery codes (hashes stay server-side).
	TOTPRecoveryLeft int       `json:"totpRecoveryLeft"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// PartnerProfile mirrors partner_profiles.
type PartnerProfile struct {
	ID                         string     `json:"id"`
	UserID                     string     `json:"userId"`
	CompanyName                string     `json:"companyName"`
	BusinessNumber             *string    `json:"businessNumber,omitempty"`
	SettlementBank             *string    `json:"settlementBank,omitempty"`
	SettlementAccountEncrypted *string    `json:"-"`
	ContactEmail               string     `json:"contactEmail"`
	Status                     string     `json:"status"`
	ApprovedAt                 *time.Time `json:"approvedAt,omitempty"`
	CreatedAt                  time.Time  `json:"createdAt"`
	UpdatedAt                  time.Time  `json:"updatedAt"`
}

// AIProviderPolicy mirrors ai_provider_policies.
type AIProviderPolicy struct {
	ID                        string     `json:"id"`
	Provider                  string     `json:"provider"`
	AllowsStandaloneStreaming bool       `json:"allowsStandaloneStreaming"`
	AllowedPlanTiers          []string   `json:"allowedPlanTiers"`
	BlockedPlanTiers          []string   `json:"blockedPlanTiers"`
	SyncOnlyPlanTiers         []string   `json:"syncOnlyPlanTiers"`
	MinBitrateKbps            int        `json:"minBitrateKbps"`
	BlockedMetadataKeywords   []string   `json:"blockedMetadataKeywords"`
	RejectionMessageKo        string     `json:"rejectionMessageKo"`
	CommercialTermsURL        *string    `json:"commercialTermsUrl,omitempty"`
	MaxGeneratedAt            *time.Time `json:"maxGeneratedAt,omitempty"`
	TermsSnapshotURL          *string    `json:"termsSnapshotUrl,omitempty"`
	TermsCheckedAt            *time.Time `json:"termsCheckedAt,omitempty"`
	CreatedAt                 time.Time  `json:"createdAt"`
	UpdatedAt                 time.Time  `json:"updatedAt"`
}

// PartnerAISubscription mirrors partner_ai_subscriptions.
type PartnerAISubscription struct {
	ID             string     `json:"id"`
	PartnerID      string     `json:"partnerId"`
	Provider       string     `json:"provider"`
	PlanName       string     `json:"planName"`
	ProofObjectKey *string    `json:"proofObjectKey,omitempty"`
	Status         string     `json:"status"`
	VerifiedBy     *string    `json:"verifiedBy,omitempty"`
	VerifiedAt     *time.Time `json:"verifiedAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// LegalTerm mirrors legal_terms.
type LegalTerm struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Version     string    `json:"version"`
	TitleKo     string    `json:"titleKo"`
	BodyMd      string    `json:"bodyMd"`
	IsRequired  bool      `json:"isRequired"`
	EffectiveAt time.Time `json:"effectiveAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// UserTermConsent mirrors user_term_consents.
type UserTermConsent struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	TermsID     string    `json:"termsId"`
	ConsentedAt time.Time `json:"consentedAt"`
	IP          *string   `json:"ip,omitempty"`
	UserAgent   *string   `json:"userAgent,omitempty"`
}

// Album mirrors albums.
type Album struct {
	ID             string     `json:"id"`
	PartnerID      string     `json:"partnerId"`
	Title          string     `json:"title"`
	CoverObjectKey *string    `json:"coverObjectKey,omitempty"`
	ReleaseType    string     `json:"releaseType"`
	Status         string     `json:"status"`
	ReleasedAt     *time.Time `json:"releasedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// Track mirrors tracks.
type Track struct {
	ID                   string    `json:"id"`
	AlbumID              *string   `json:"albumId,omitempty"`
	PartnerID            string    `json:"partnerId"`
	Title                string    `json:"title"`
	Genre                *string   `json:"genre,omitempty"`
	MoodTags             []string  `json:"moodTags,omitempty"`
	AIGenerationType     string    `json:"aiGenerationType"`
	AICreditParts        []string  `json:"aiCreditParts,omitempty"`
	AIModelVersion       *string   `json:"aiModelVersion,omitempty"`
	AIPrompt             *string   `json:"aiPrompt,omitempty"`
	IsPromptPublic       bool      `json:"isPromptPublic"`
	IsInstrumental       bool      `json:"isInstrumental"`
	Lyrics               *string   `json:"lyrics,omitempty"`
	DurationMs           *int      `json:"durationMs,omitempty"`
	OriginalObjectKey    string    `json:"originalObjectKey,omitempty"`
	AudioFingerprintHash *string   `json:"audioFingerprintHash,omitempty"`
	HLSPrefix            *string   `json:"hlsPrefix,omitempty"`
	WaveformJSON         *string   `json:"waveformJson,omitempty"`
	AIProvider           string    `json:"aiProvider"`
	AIGenerated          bool      `json:"aiGenerated"`
	AIDisclosureText     *string   `json:"aiDisclosureText,omitempty"`
	BitrateKbps          *int      `json:"bitrateKbps,omitempty"`
	ISRC                 *string   `json:"isrc,omitempty"`
	BPM                  *float64  `json:"bpm,omitempty"`
	MusicalKey           *string   `json:"musicalKey,omitempty"`
	Status               string    `json:"status"`
	RejectionReasonKo    *string   `json:"rejectionReasonKo,omitempty"`
	PlayCount            int64     `json:"playCount"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// PublicView returns a copy safe for anonymous and listener responses (B-05):
// the master storage key, fingerprint, HLS prefix and review notes are
// internal, and a prompt the partner kept private is withheld.
func (t *Track) PublicView() *Track {
	if t == nil {
		return nil
	}
	c := *t
	c.OriginalObjectKey = ""
	c.AudioFingerprintHash = nil
	c.HLSPrefix = nil
	c.RejectionReasonKo = nil
	if !c.IsPromptPublic {
		c.AIPrompt = nil
	}
	return &c
}

// TrackRightsDeclaration mirrors track_rights_declarations.
type TrackRightsDeclaration struct {
	ID                              string     `json:"id"`
	TrackID                         string     `json:"trackId"`
	PartnerID                       string     `json:"partnerId"`
	DeclaresOwnership               bool       `json:"declaresOwnership"`
	DeclaresNoVoiceClone            bool       `json:"declaresNoVoiceClone"`
	DeclaresTrainingConsent         bool       `json:"declaresTrainingConsent"`
	ConfirmedNoUnauthorizedSampling bool       `json:"confirmedNoUnauthorizedSampling"`
	ConfirmedIndemnification        bool       `json:"confirmedIndemnification"`
	PlanTierUsed                    *string    `json:"planTierUsed,omitempty"`
	ProviderTrackURL                *string    `json:"providerTrackUrl,omitempty"`
	HumanContributionTypes          []string   `json:"humanContributionTypes,omitempty"`
	HumanContributionDesc           *string    `json:"humanContributionDesc,omitempty"`
	Gate1PolicyPassed               bool       `json:"gate1PolicyPassed"`
	Gate2DatePassed                 bool       `json:"gate2DatePassed"`
	Gate3FFprobePassed              bool       `json:"gate3FfprobePassed"`
	FFprobeReportJSON               *string    `json:"ffprobeReportJson,omitempty"`
	SignedIP                        *string    `json:"signedIp,omitempty"`
	SourceModel                     *string    `json:"sourceModel,omitempty"`
	SubscriptionID                  *string    `json:"subscriptionId,omitempty"`
	GeneratedAt                     *time.Time `json:"generatedAt,omitempty"`
	CollectiveTrustStatus           string     `json:"collectiveTrustStatus"`
	CollectiveName                  *string    `json:"collectiveName,omitempty"`
	CertSnapshotJSON                *string    `json:"certSnapshotJson,omitempty"`
	CertSnapshotAt                  *time.Time `json:"certSnapshotAt,omitempty"`
	DeclaredAt                      time.Time  `json:"declaredAt"`
}

// TrackCertificate mirrors track_certificates: the public "상업적 이용권
// 보유 인증" credential for one READY track (P2-6 #24).
type TrackCertificate struct {
	ID         string    `json:"id"`
	TrackID    string    `json:"trackId"`
	CertNumber string    `json:"certNumber"`
	IssuedBy   *string   `json:"issuedBy,omitempty"`
	IssuedAt   time.Time `json:"issuedAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TranscodingJob mirrors transcoding_jobs.
type TranscodingJob struct {
	ID             string     `json:"id"`
	TrackID        string     `json:"trackId"`
	Status         string     `json:"status"`
	InputObjectKey string     `json:"inputObjectKey"`
	OutputPrefix   *string    `json:"outputPrefix,omitempty"`
	BitrateKbps    *int       `json:"bitrateKbps,omitempty"`
	Error          *string    `json:"error,omitempty"`
	Attempts       int        `json:"attempts"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// GenerationJob mirrors generation_jobs (WI-03). Statuses reuse the
// transcoding job constants (QUEUED/RUNNING/SUCCEEDED/FAILED).
type GenerationJob struct {
	ID              string    `json:"id"`
	PartnerID       string    `json:"partnerId"`
	Prompt          string    `json:"prompt"`
	Lyrics          *string   `json:"lyrics,omitempty"`
	Model           string    `json:"model"`
	LicenseRef      string    `json:"licenseRef,omitempty"`
	ParamsJSON      string    `json:"paramsJson"`
	Status          string    `json:"status"`
	OutputObjectKey *string   `json:"outputObjectKey,omitempty"`
	DurationMs      *int      `json:"durationMs,omitempty"`
	Error           *string   `json:"error,omitempty"`
	Attempts        int       `json:"attempts"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Store business/device statuses (WI-05).
const (
	StorePending   = "PENDING"
	StoreApproved  = "APPROVED"
	StoreSuspended = "SUSPENDED"
	DevicePending  = "PENDING"
	DeviceActive   = "ACTIVE"
	DeviceRevoked  = "REVOKED"
)

// StoreBusiness mirrors store_businesses: one store = one license.
type StoreBusiness struct {
	ID             string     `json:"id"`
	PartnerID      string     `json:"partnerId"`
	BusinessNumber string     `json:"businessNumber"`
	StoreName      string     `json:"storeName"`
	LicenseCount   int        `json:"licenseCount"`
	Status         string     `json:"status"`
	VerifiedBy     *string    `json:"verifiedBy,omitempty"`
	VerifiedAt     *time.Time `json:"verifiedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// StoreDevice mirrors store_devices: a certified player terminal.
type StoreDevice struct {
	ID               string     `json:"id"`
	BusinessID       string     `json:"businessId"`
	DeviceName       string     `json:"deviceName"`
	PairingCode      *string    `json:"pairingCode,omitempty"`
	PairingExpiresAt *time.Time `json:"pairingExpiresAt,omitempty"`
	DeviceTokenHash  *string    `json:"-"`
	Status           string     `json:"status"`
	Volume           int        `json:"volume"`
	LastSeenAt       *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// StoreSchedule mirrors store_schedules: weekday 0=Sunday.
type StoreSchedule struct {
	ID         string    `json:"id"`
	BusinessID string    `json:"businessId"`
	Weekday    int       `json:"weekday"`
	OpenTime   string    `json:"openTime"`
	CloseTime  string    `json:"closeTime"`
	PlaylistID *string   `json:"playlistId,omitempty"`
	IsClosed   bool      `json:"isClosed"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// StorePlayLog mirrors store_play_logs.
type StorePlayLog struct {
	ID                 string    `json:"id"`
	DeviceID           string    `json:"deviceId"`
	TrackID            string    `json:"trackId"`
	PlayedAt           time.Time `json:"playedAt"`
	DurationPlayedSecs int       `json:"durationPlayedSecs"`
	CountedAsPlay      bool      `json:"countedAsPlay"`
	CreatedAt          time.Time `json:"createdAt"`
}

// CopyrightDispute mirrors copyright_disputes.
type CopyrightDispute struct {
	ID                  string     `json:"id"`
	TrackID             string     `json:"trackId"`
	ReporterUserID      *string    `json:"reporterUserId,omitempty"`
	Reason              string     `json:"reason"`
	Description         string     `json:"description"`
	Status              string     `json:"status"`
	StrikeApplied       bool       `json:"strikeApplied"`
	ResolvedBy          *string    `json:"resolvedBy,omitempty"`
	ResolvedAt          *time.Time `json:"resolvedAt,omitempty"`
	TakedownRequestedAt *time.Time `json:"takedownRequestedAt,omitempty"`
	PartnerNotifiedAt   *time.Time `json:"partnerNotifiedAt,omitempty"`
	ResumeRequestedAt   *time.Time `json:"resumeRequestedAt,omitempty"`
	ResumeScheduledAt   *time.Time `json:"resumeScheduledAt,omitempty"`
	ResumedAt           *time.Time `json:"resumedAt,omitempty"`
	SLANotifiedAt       *time.Time `json:"slaNotifiedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// AuditEvent mirrors audit_events (append-only admin action trail).
type AuditEvent struct {
	ID          string    `json:"id"`
	ActorUserID *string   `json:"actorUserId,omitempty"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entityType,omitempty"`
	EntityID    string    `json:"entityId,omitempty"`
	Metadata    string    `json:"metadata,omitempty"`
	IP          string    `json:"ip,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Playlist mirrors playlists.
type Playlist struct {
	ID                 string    `json:"id"`
	OwnerUserID        string    `json:"ownerUserId"`
	Title              string    `json:"title"`
	Description        *string   `json:"description,omitempty"`
	IsPublic           bool      `json:"isPublic"`
	IsOfficialCuration bool      `json:"isOfficialCuration"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// PlaylistTrack mirrors playlist_tracks.
type PlaylistTrack struct {
	ID         string    `json:"id"`
	PlaylistID string    `json:"playlistId"`
	TrackID    string    `json:"trackId"`
	Position   int       `json:"position"`
	AddedAt    time.Time `json:"addedAt"`
}

// UserPlaybackState mirrors user_playback_states.
type UserPlaybackState struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	TrackID    string    `json:"trackId"`
	PositionMs int       `json:"positionMs"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// StreamingLog mirrors streaming_logs.
type StreamingLog struct {
	ID            string    `json:"id"`
	UserID        *string   `json:"userId,omitempty"`
	TrackID       *string   `json:"trackId,omitempty"`
	PartnerID     *string   `json:"partnerId,omitempty"`
	PlayedAt      time.Time `json:"playedAt"`
	PlayMs        int       `json:"playMs"`
	CountedAsPlay bool      `json:"countedAsPlay"`
	IP            *string   `json:"ip,omitempty"`
	UserAgent     *string   `json:"userAgent,omitempty"`
	HLSTokenJTI   *string   `json:"hlsTokenJti,omitempty"`
}

// PartnerSettlement mirrors partner_settlements.
//
// Canon (방향결정서 §5-1): 유통 순매출 70:30 (파트너 70). GrossAmountKRW
// is the partner's 70% share of attributed revenue (pool share + direct),
// never a per-play payout. StorePoolKRW/DirectKRW are the admin-attested
// revenue inputs; StorePlaySecs/TotalStorePlaySecs are the time-pro-rata
// basis (qualified 30s+ store plays, APPROVED businesses/ACTIVE devices).
// PlayCount keeps the qualifying store play count for the same window.
type PartnerSettlement struct {
	ID                 string     `json:"id"`
	PartnerID          string     `json:"partnerId"`
	PeriodStart        time.Time  `json:"periodStart"`
	PeriodEnd          time.Time  `json:"periodEnd"`
	GrossAmountKRW     int64      `json:"grossAmountKrw"`
	WithholdingKRW     int64      `json:"withholdingKrw"`
	NetAmountKRW       int64      `json:"netAmountKrw"`
	PlayCount          int64      `json:"playCount"`
	StorePoolKRW       int64      `json:"storePoolKrw"`
	DirectKRW          int64      `json:"directKrw"`
	StorePlaySecs      int64      `json:"storePlaySecs"`
	TotalStorePlaySecs int64      `json:"totalStorePlaySecs"`
	Status             string     `json:"status"`
	HoldReason         *string    `json:"holdReason,omitempty"`
	HeldAt             *time.Time `json:"heldAt,omitempty"`
	HeldBy             *string    `json:"heldBy,omitempty"`
	ForfeitureReason   *string    `json:"forfeitureReason,omitempty"`
	IncomeType         string     `json:"incomeType"`
	PaidAt             *time.Time `json:"paidAt,omitempty"`
	PayoutReference    *string    `json:"payoutReference,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// Notification kinds (notifications.kind).
const (
	NotifyDisputeTakedown    = "DISPUTE_TAKEDOWN"
	NotifyDisputeCounter     = "DISPUTE_COUNTER"
	NotifyDisputeResolved    = "DISPUTE_RESOLVED"
	NotifySettlementCalc     = "SETTLEMENT_CALCULATED"
	NotifySettlementApproved = "SETTLEMENT_APPROVED"
	NotifySettlementPaid     = "SETTLEMENT_PAID"
	NotifySubVerified        = "SUBSCRIPTION_VERIFIED"
	NotifySubRejected        = "SUBSCRIPTION_REJECTED"
	NotifyNotice             = "NOTICE"
)

// Notification mirrors the notifications table.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Kind      string    `json:"kind"`
	TitleKo   string    `json:"titleKo"`
	BodyKo    string    `json:"bodyKo"`
	LinkPath  string    `json:"linkPath"`
	IsRead    bool      `json:"isRead"`
	CreatedAt time.Time `json:"createdAt"`
}

// Admin action journal verbs (admin_action_logs.action).
const (
	AuditReviewTrack        = "REVIEW_TRACK"
	AuditReviewPartner      = "REVIEW_PARTNER"
	AuditReviewSubscription = "REVIEW_SUBSCRIPTION"
	AuditPolicyUpdate       = "POLICY_UPDATE"
	AuditDisputeResolve     = "DISPUTE_RESOLVE"
	AuditTakedown           = "TAKEDOWN"
	AuditSettleCalculate    = "SETTLEMENT_CALCULATE"
	AuditSettleApprove      = "SETTLEMENT_APPROVE"
	AuditSettleMarkPaid     = "SETTLEMENT_MARK_PAID"
	AuditSettleHold         = "SETTLEMENT_HOLD"
	AuditSettleRelease      = "SETTLEMENT_RELEASE"
	AuditUserSetActive      = "USER_SET_ACTIVE"
	AuditUserSetRole        = "USER_SET_ROLE"
	AuditCurationCreate     = "CURATION_CREATE"
)

// AdminActionLog mirrors admin_action_logs.
type AdminActionLog struct {
	ID          string    `json:"id"`
	AdminUserID *string   `json:"adminUserId,omitempty"`
	Action      string    `json:"action"`
	TargetType  string    `json:"targetType"`
	TargetID    string    `json:"targetId"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"createdAt"`
}

// UserBlock mirrors user_blocks.
type UserBlock struct {
	ID           string     `json:"id"`
	UserID       string     `json:"userId"`
	BlockedBy    *string    `json:"blockedBy,omitempty"`
	Reason       string     `json:"reason"`
	StrikeNumber int        `json:"strikeNumber"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// UserArtistBlock mirrors user_artist_blocks (UGC artist block).
type UserArtistBlock struct {
	ID               string    `json:"id"`
	BlockerUserID    string    `json:"blockerUserId"`
	BlockedPartnerID string    `json:"blockedPartnerId"`
	Reason           string    `json:"reason"`
	CreatedAt        time.Time `json:"createdAt"`
}
