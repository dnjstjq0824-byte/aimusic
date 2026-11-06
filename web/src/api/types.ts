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
  originalObjectKey: string;
  hlsPrefix?: string | null;
  aiProvider: string;
  aiGenerated: boolean;
  aiGenerationType?: AIDisclosure | string;
  aiModelVersion?: string | null;
  aiPrompt?: string | null;
  isPromptPublic?: boolean;
  isInstrumental?: boolean;
  lyrics?: string | null;
  aiDisclosureText?: string | null;
  audioFingerprintHash?: string | null;
  waveformJson?: string | null;
  bitrateKbps?: number | null;
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
  positionMs: number;
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
  status: string;
  forfeitureReason?: string | null;
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
