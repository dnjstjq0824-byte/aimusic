import { api, unwrap } from './client';
import type {
  AIDisclosure,
  Album,
  CopyrightDispute,
  FinalizeOutput,
  HeartbeatOutput,
  LegalTerm,
  PartnerAISubscription,
  PartnerProfile,
  PartnerSettlement,
  PlaybackState,
  Playlist,
  PlaylistTrack,
  PrecheckOutput,
  ReviewQueues,
  Role,
  StreamGrant,
  Tokens,
  Track,
  User,
  UserArtistBlock,
  UserBlock,
  UserTermConsent,
  AIProviderPolicy,
} from './types';

// Thin wrappers over the exact router.go URIs.

export const authApi = {
  register(email: string, password: string, displayName: string) {
    return unwrap<User>(api.post('/auth/register', { email, password, displayName }));
  },
  registerPartner(input: { email: string; password: string; displayName: string; company: string; contactEmail?: string }) {
    return unwrap<{ user: User; partner: PartnerProfile }>(api.post('/auth/register/partner', input));
  },
  login(email: string, password: string) {
    return unwrap<{ tokens: Tokens; user: User }>(api.post('/auth/login', { email, password }));
  },
  me() {
    return unwrap<{ userId: string; role: Role }>(api.get('/auth/me'));
  },
  withdraw() {
    return unwrap<{ withdrawn: boolean }>(api.post('/auth/withdraw'));
  },
};

export const legalApi = {
  listTerms() {
    return unwrap<LegalTerm[]>(api.get('/legal/terms'));
  },
  getTerm(code: string) {
    return unwrap<LegalTerm>(api.get(`/legal/terms/${encodeURIComponent(code)}`));
  },
  consent(termsId: string) {
    return unwrap<{ consented: boolean }>(api.post('/legal/consents', { termsId }));
  },
  myConsents() {
    return unwrap<UserTermConsent[]>(api.get('/legal/consents/me'));
  },
};

export const partnerApi = {
  listPolicies() {
    return unwrap<AIProviderPolicy[]>(api.get('/partner/license-policies'));
  },
  getPolicy(provider: string) {
    return unwrap<AIProviderPolicy>(api.get(`/partner/license-policies/${encodeURIComponent(provider)}`));
  },
  listSubscriptions() {
    return unwrap<PartnerAISubscription[]>(api.get('/partner/subscriptions'));
  },
  createSubscription(input: { provider: string; planName: string; proofObjectKey?: string }) {
    return unwrap<PartnerAISubscription>(api.post('/partner/subscriptions', input));
  },
  preCheck(input: {
    provider: string;
    planTierUsed?: string;
    audioSha256: string;
    fileName: string;
    generatedAt: string;
    subscriptionId?: string;
    requiresActiveSub?: boolean;
  }) {
    return unwrap<PrecheckOutput>(api.post('/partner/pre-check', input));
  },
  finalize(input: {
    stagingKey: string;
    title: string;
    provider: string;
    planTierUsed?: string;
    audioSha256: string;
    generatedAt: string;
    subscriptionId?: string;
    albumId?: string;
    genre?: string;
    moodTags?: string[];
    aiGenerationType?: AIDisclosure | string;
    aiModelVersion?: string;
    aiPrompt?: string;
    isPromptPublic?: boolean;
    isInstrumental?: boolean;
    lyrics?: string;
    durationMs?: number;
    bitrateKbps?: number;
    disclosure?: AIDisclosure | string;
    rights: {
      declaresOwnership: boolean;
      declaresNoVoiceClone: boolean;
      declaresTrainingConsent: boolean;
      confirmedNoUnauthorizedSampling?: boolean;
      confirmedIndemnification?: boolean;
      planTierUsed?: string;
      providerTrackUrl?: string;
      humanContributionTypes?: string[];
      humanContributionDesc?: string;
      sourceModel?: string;
      subscriptionId?: string;
    };
  }) {
    return unwrap<FinalizeOutput>(api.post('/partner/finalize', input));
  },
  createAlbum(input: { title: string; releaseType?: string; coverObjectKey?: string }) {
    return unwrap<Album>(api.post('/partner/albums', input));
  },
  listAlbums(limit = 20, offset = 0) {
    return unwrap<Album[]>(api.get('/partner/albums', { params: { limit, offset } }));
  },
  listMyTracks(limit = 20, offset = 0) {
    return unwrap<Track[]>(api.get('/partner/tracks', { params: { limit, offset } }));
  },
};

export const trackApi = {
  list(limit = 20, offset = 0, filters?: { provider?: string; genre?: string; mood?: string }) {
    return unwrap<Track[]>(api.get('/tracks', { params: { limit, offset, ...filters } }));
  },
  get(id: string) {
    return unwrap<Track>(api.get(`/tracks/${encodeURIComponent(id)}`));
  },
  stream(id: string) {
    return unwrap<StreamGrant>(api.get(`/tracks/${encodeURIComponent(id)}/stream`));
  },
};

export const playbackApi = {
  heartbeat(input: { trackId: string; positionMs: number; playMs: number; jti?: string }) {
    return unwrap<HeartbeatOutput>(api.post('/playback/heartbeat', input));
  },
  getState(trackId: string) {
    return unwrap<PlaybackState>(api.get(`/playback/state/${encodeURIComponent(trackId)}`));
  },
  setState(trackId: string, positionMs: number) {
    return unwrap<PlaybackState>(api.put(`/playback/state/${encodeURIComponent(trackId)}`, { positionMs }));
  },
};

export const playlistApi = {
  listMine() {
    return unwrap<Playlist[]>(api.get('/playlists'));
  },
  create(title: string, isPublic = false) {
    return unwrap<Playlist>(api.post('/playlists', { title, isPublic }));
  },
  get(id: string) {
    return unwrap<Playlist>(api.get(`/playlists/${encodeURIComponent(id)}`));
  },
  update(id: string, title: string, isPublic: boolean) {
    return unwrap<Playlist>(api.put(`/playlists/${encodeURIComponent(id)}`, { title, isPublic }));
  },
  remove(id: string) {
    return unwrap<{ deleted: boolean }>(api.delete(`/playlists/${encodeURIComponent(id)}`));
  },
  listTracks(id: string) {
    return unwrap<PlaylistTrack[]>(api.get(`/playlists/${encodeURIComponent(id)}/tracks`));
  },
  addTrack(id: string, trackId: string) {
    return unwrap<{ added: boolean }>(api.post(`/playlists/${encodeURIComponent(id)}/tracks`, { trackId }));
  },
  removeTrack(id: string, trackId: string) {
    return unwrap<{ removed: boolean }>(api.delete(`/playlists/${encodeURIComponent(id)}/tracks/${encodeURIComponent(trackId)}`));
  },
  reorder(id: string, trackIds: string[]) {
    return unwrap<{ reordered: boolean }>(api.put(`/playlists/${encodeURIComponent(id)}/reorder`, { trackIds }));
  },
};

export const disputeApi = {
  report(trackId: string, reason: string, description?: string) {
    return unwrap<CopyrightDispute>(api.post('/disputes/report', { trackId, reason, description }));
  },
  counter(id: string, note: string) {
    return unwrap<CopyrightDispute>(api.post(`/disputes/${encodeURIComponent(id)}/counter`, { note }));
  },
  myBlocks() {
    return unwrap<UserBlock[]>(api.get('/disputes/blocks'));
  },
};

export const userBlocksApi = {
  blockArtist(partnerId: string, reason?: string) {
    return unwrap<UserArtistBlock>(api.post('/users/blocks', { partnerId, reason }));
  },
  listArtistBlocks() {
    return unwrap<UserArtistBlock[]>(api.get('/users/blocks'));
  },
  unblockArtist(partnerId: string) {
    return unwrap<{ unblocked: boolean }>(api.delete(`/users/blocks/${encodeURIComponent(partnerId)}`));
  },
};

export const adminApi = {
  pending() {
    return unwrap<ReviewQueues>(api.get('/admin/pending'));
  },
  pendingReviews() {
    return unwrap<ReviewQueues>(api.get('/admin/pending'));
  },
  reviewTrack(trackId: string, approve: boolean, reason?: string) {
    return unwrap<Track>(api.post('/admin/review', { trackId, approve, reason }));
  },
  reviewPartner(partnerId: string, approve: boolean) {
    return unwrap<PartnerProfile>(api.post('/admin/review/partner', { partnerId, approve }));
  },
  reviewSubscription(subscriptionId: string, approve: boolean, expiresAt?: string) {
    return unwrap<PartnerAISubscription>(
      api.post('/admin/review/subscription', { subscriptionId, approve, expiresAt }),
    );
  },
  listPolicies() {
    return unwrap<AIProviderPolicy[]>(api.get('/admin/policies'));
  },
  updatePolicy(
    provider: string,
    input: {
      allowsStandaloneStreaming: boolean;
      allowedPlanTiers?: string[];
      blockedPlanTiers?: string[];
      syncOnlyPlanTiers?: string[];
      minBitrateKbps: number;
      blockedMetadataKeywords: string[];
      rejectionMessageKo: string;
      commercialTermsUrl?: string | null;
    },
  ) {
    return unwrap<AIProviderPolicy>(api.put(`/admin/policies/${encodeURIComponent(provider)}`, input));
  },
  listDisputes(limit = 20, offset = 0) {
    return unwrap<CopyrightDispute[]>(api.get('/admin/disputes', { params: { limit, offset } }));
  },
  resolveDispute(id: string, uphold: boolean) {
    return unwrap<CopyrightDispute>(api.post(`/admin/disputes/${encodeURIComponent(id)}/resolve`, { uphold }));
  },
  curation(limit = 20) {
    return unwrap<Track[]>(api.get('/admin/curation', { params: { limit } }));
  },
  createCurationPlaylist(title: string, description?: string, trackIds: string[] = []) {
    return unwrap<Playlist>(api.post('/admin/curation/playlists', { title, description, trackIds }));
  },
  takedown(trackId: string, reason?: string) {
    return unwrap<Track>(api.post('/admin/curation/takedown', { trackId, reason }));
  },
  calculateSettlement(partnerId: string, periodStart: string, periodEnd: string, ratePerPlayKrw?: number) {
    return unwrap<PartnerSettlement>(
      api.post('/admin/settlements/calculate', { partnerId, periodStart, periodEnd, ratePerPlayKrw }),
    );
  },
  listSettlements(partnerId: string) {
    return unwrap<PartnerSettlement[]>(api.get(`/admin/settlements/${encodeURIComponent(partnerId)}`));
  },
  approveSettlement(partnerId: string, periodStart: string, periodEnd: string) {
    return unwrap<PartnerSettlement>(api.post('/admin/settlements/approve', { partnerId, periodStart, periodEnd }));
  },
};
