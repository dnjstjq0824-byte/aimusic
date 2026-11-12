// Domain types mirroring backend/internal/domain/entities.go JSON tags
// and handler request/response shapes in backend/internal/handler/*.go.

export type Role = 'ADMIN' | 'PARTNER' | 'USER';

export interface Tokens {
  accessToken: string;
  refreshToken: string;
  expiresInSecs: number;
}

export interface User {
  id: string;
  email: string;
  role: Role;
  displayName: string;
  isActive: boolean;
  birthYear?: number | null;
  ageConfirmedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface PartnerProfile {
  id: string;
  userId: string;
  companyName: string;
  businessNumber?: string | null;
  contactEmail: string;
  status: 'PENDING' | 'APPROVED' | 'SUSPENDED' | 'TERMINATED';
  approvedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface AIProviderPolicy {
  id: string;
  provider: string;
  allowsStandaloneStreaming: boolean;
  allowedPlanTiers?: string[];
  blockedPlanTiers?: string[];
  syncOnlyPlanTiers?: string[];
  minBitrateKbps: number;
  blockedMetadataKeywords: string[];
  rejectionMessageKo: string;
  commercialTermsUrl?: string | null;
  maxGeneratedAt?: string | null;
  termsSnapshotUrl?: string | null;
  termsCheckedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface PartnerAISubscription {
  id: string;
  partnerId: string;
  provider: string;
  planName: string;
  proofObjectKey?: string | null;
  status: 'PENDING' | 'VERIFIED' | 'REJECTED' | 'EXPIRED';
  verifiedBy?: string | null;
  verifiedAt?: string | null;
  expiresAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export type AIDisclosure = 'FULL_AI' | 'AI_ASSISTED_HUMAN';

export interface Track {
  id: string;
  albumId?: string | null;
  partnerId: string;
  title: string;
  genre?: string | null;
  moodTags?: string[];
  durationMs?: number | null;
  originalObjectKey?: string;
  hlsPrefix?: string | null;
  aiProvider: string;
  aiGenerated: boolean;
  aiGenerationType?: AIDisclosure | string;
  aiCreditParts?: string[];
  aiModelVersion?: string | null;
  aiPrompt?: string | null;
  isPromptPublic?: boolean;
  isInstrumental?: boolean;
  lyrics?: string | null;
  aiDisclosureText?: string | null;
  audioFingerprintHash?: string | null;
  waveformJson?: string | null;
  bitrateKbps?: number | null;
  isrc?: string | null;
  bpm?: number | null;
  musicalKey?: string | null;
  status: string;
  rejectionReasonKo?: string | null;
  playCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface TrackRightsDeclaration {
  id: string;
  trackId: string;
  partnerId: string;
  declaresOwnership: boolean;
  declaresNoVoiceClone: boolean;
  declaresTrainingConsent: boolean;
  confirmedNoUnauthorizedSampling: boolean;
  confirmedIndemnification: boolean;
  planTierUsed?: string | null;
  providerTrackUrl?: string | null;
  humanContributionTypes?: string[];
  humanContributionDesc?: string | null;
  gate1PolicyPassed: boolean;
  gate2DatePassed: boolean;
  gate3FfprobePassed?: boolean | null;
  ffprobeReportJson?: string | null;
  signedIp?: string | null;
  sourceModel?: string | null;
  subscriptionId?: string | null;
  generatedAt: string;
  collectiveTrustStatus?: string;
  collectiveName?: string | null;
  createdAt: string;
}

export interface UserArtistBlock {
  id: string;
  blockerUserId: string;
  blockedPartnerId: string;
  reason: string;
  createdAt: string;
}

export interface Album {
  id: string;
  partnerId: string;
  title: string;
  coverObjectKey?: string | null;
  releaseType: string;
  status: string;
  releasedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Playlist {
  id: string;
  ownerUserId: string;
  title: string;
  isPublic: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface PlaylistTrack {
  id: string;
  playlistId: string;
  trackId: string;
  position: number;
  addedAt: string;
}

export interface StoreBusiness {
  id: string;
  partnerId: string;
  businessNumber: string;
  storeName: string;
  licenseCount: number;
  status: string;
  verifiedBy?: string | null;
  verifiedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface StoreDevice {
  id: string;
  businessId: string;
  deviceName: string;
  pairingCode?: string | null;
  pairingExpiresAt?: string | null;
  status: string;
  volume: number;
  lastSeenAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface StoreSchedule {
  id: string;
  businessId: string;
  weekday: number;
  openTime: string;
  closeTime: string;
  playlistId?: string | null;
  isClosed: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface StorePlayLog {
  id: string;
  deviceId: string;
  trackId: string;
  playedAt: string;
  durationPlayedSecs: number;
  countedAsPlay: boolean;
  createdAt: string;
}

export interface DeviceQueueTrack {
  trackId: string;
  title: string;
  durationMs?: number | null;
}

export interface DeviceQueue {
  businessId: string;
  businessName: string;
  open: boolean;
  reason: string;
  playlistId?: string | null;
  playlistName?: string;
  volume: number;
  tracks: DeviceQueueTrack[];
}

export interface DSPKitTrack {
  trackId: string;
  title: string;
  artistName: string;
  albumTitle?: string;
  genre?: string;
  moodTags?: string[];
  durationMs?: number;
  isrc: string;
  bpm?: number | null;
  musicalKey?: string;
  aiProvider: string;
  aiGenerated: boolean;
  aiDisclosureText?: string;
  planTierUsed?: string;
  sourceModel?: string;
  certPolicyVersion?: string;
  masterObjectKey: string;
  masterDownloadUrl: string;
  masterUrlExpiry: string;
}

export interface DSPKit {
  generatedAt: string;
  tracks: DSPKitTrack[];
  skipped: { trackId: string; reason: string }[];
}

export interface CopyrightDispute {
  id: string;
  trackId: string;
  reporterUserId?: string | null;
  reason: string;
  description: string;
  status: string;
  strikeApplied: boolean;
  resolvedBy?: string | null;
  resolvedAt?: string | null;
  takedownRequestedAt?: string | null;
  partnerNotifiedAt?: string | null;
  resumeRequestedAt?: string | null;
  resumeScheduledAt?: string | null;
  resumedAt?: string | null;
  slaNotifiedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface UserBlock {
  id: string;
  userId: string;
  blockedBy?: string | null;
  reason: string;
  strikeNumber: number;
  expiresAt?: string | null;
  createdAt: string;
}

export interface LegalTerm {
  id: string;
  code: string;
  version: string;
  titleKo: string;
  bodyMd: string;
  isRequired: boolean;
  effectiveAt: string;
  createdAt: string;
}

export interface UserTermConsent {
  id: string;
  userId: string;
  termsId: string;
  consentedAt: string;
  ip?: string | null;
  userAgent?: string | null;
}

export interface PrecheckOutput {
  eligible: boolean;
  stagingKey: string;
  uploadUrl: string;
  expiresInSecs: number;
}

export interface FinalizeOutput {
  trackId: string;
  jobId: string;
}

export interface StreamGrant {
  url: string;
  expiresAt: string;
  jti: string;
  /** HQ (256k master, pass holders) or SQ (128k, free tier). */
  quality: 'HQ' | 'SQ';
}

export interface PlaybackState {
  id: string;
  userId: string;
  trackId: string;
  positionMs: number;
  createdAt: string;
  updatedAt: string;
}

export interface HeartbeatOutput {
  countedAsPlay: boolean;
  fraudSuspected: boolean;
}

export interface PartnerSettlement {
  id: string;
  partnerId: string;
  periodStart: string;
  periodEnd: string;
  grossAmountKrw: number;
  withholdingKrw: number;
  netAmountKrw: number;
  playCount: number;
  storePoolKrw: number;
  directKrw: number;
  storePlaySecs: number;
  totalStorePlaySecs: number;
  status: string;
  holdReason?: string | null;
  heldAt?: string | null;
  heldBy?: string | null;
  forfeitureReason?: string | null;
  incomeType: string;
  createdAt: string;
  updatedAt: string;
}

export interface TrackCertificate {
  id: string;
  trackId: string;
  certNumber: string;
  issuedBy?: string | null;
  issuedAt: string;
  createdAt: string;
}

export interface CertificateView {
  certificate: TrackCertificate;
  trackId: string;
  title: string;
  aiProvider: string;
  aiModel?: string | null;
  isrc?: string | null;
  status: string;
  planTier?: string | null;
  generatedAt?: string | null;
  gatesPassed: boolean;
}

export interface GenerationJob {
  id: string;
  partnerId: string;
  prompt: string;
  lyrics?: string | null;
  model: string;
  licenseRef?: string;
  paramsJson: string;
  status: string;
  outputObjectKey?: string | null;
  durationMs?: number | null;
  error?: string | null;
  attempts: number;
  createdAt: string;
  updatedAt: string;
}

export interface ReviewQueues {
  tracks?: Track[];
  licenseGateTracks?: Track[];
  rightsByTrack?: Record<string, TrackRightsDeclaration>;
  subsById?: Record<string, PartnerAISubscription>;
  partners?: PartnerProfile[];
  subscriptions?: PartnerAISubscription[];
}

export interface ApiErrorBody {
  error: { code: string; message: string };
}

// One page of a paged listing (mirrors the backend page envelope).
export interface Page<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

export type TrackPage = Page<Track>;

export type TrackSort = 'recent' | 'oldest' | 'popular' | 'title';

export interface OAuthProviderInfo {
  provider: string;
  enabled: boolean;
}

export interface BillingPlan {
  id: string;
  code: string;
  nameKo: string;
  amountKrw: number;
  interval: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface BillingOrder {
  id: string;
  userId: string;
  planId: string;
  orderId: string;
  amountKrw: number;
  status: string;
  tossPaymentKey?: string;
  approvedAt?: string;
  refundConsentAt?: string | null;
  refundTermsVersion?: string | null;
  refundedAmountKrw?: number | null;
  refundedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export type CancelMode = 'END_OF_PERIOD' | 'IMMEDIATE';

/** Server-computed refund for ending the active pass now (billing.go ComputeRefund). */
export interface RefundQuote {
  type: 'WITHDRAWAL' | 'PRORATED';
  amountKrw: number;
  refundKrw: number;
  usedDays: number;
  totalDays: number;
  paidAt: string;
  periodEnd: string;
  hasUsage: boolean;
  cancelRequested: boolean;
  firstUsageAt?: string | null;
}

export interface CancelResult {
  mode: CancelMode;
  quote?: RefundQuote;
  order?: BillingOrder;
  subscription?: BillingSubscription;
}

export interface BillingSubscription {
  id: string;
  userId: string;
  planId: string;
  orderId?: string;
  status: string;
  currentPeriodStart: string;
  currentPeriodEnd: string;
  /** Set by 일반해지: access continues until currentPeriodEnd. */
  cancelRequestedAt?: string | null;
  plan?: BillingPlan;
  createdAt: string;
  updatedAt: string;
}

export interface AdminOverview {
  users: { total: number; admins: number; partners: number; users: number };
  tracks: { ready: number; pendingGate: number; takenDown: number };
  reviews: { tracks: number; partners: number; subscriptions: number };
  disputesOpen: number;
  plays: { last24h: number; total: number };
  queueDepth: number | null;
  jobsFailed24h: number;
  settlementsPending: { count: number; netKrw: number };
}

export interface AuditEvent {
  id: string;
  actorUserId?: string;
  action: string;
  entityType?: string;
  entityId?: string;
  metadata?: string;
  ip?: string;
  createdAt: string;
}
