import { api, deviceApiClient, unwrap } from './client';
import type {
  AdminOverview,
  AuditEvent,
  AIDisclosure,
  Album,
  BillingOrder,
  BillingPlan,
  BillingSubscription,
  CertificateView,
  CopyrightDispute,
  DeviceQueue,
  DSPKit,
  FinalizeOutput,
  GenerationJob,
  HeartbeatOutput,
  LegalTerm,
  OAuthProviderInfo,
  PartnerAISubscription,
  PartnerProfile,
  PartnerSettlement,
  PlaybackState,
  Playlist,
  PlaylistTrack,
  PrecheckOutput,
  ReviewQueues,
  Role,
  StoreBusiness,
  StoreDevice,
  StorePlayLog,
  StoreSchedule,
  StreamGrant,
  Tokens,
  Track,
  TrackCertificate,
  TrackPage,
  User,
  UserArtistBlock,
  UserBlock,
  UserTermConsent,
  AIProviderPolicy,
} from './types';

// Thin wrappers over the exact router.go URIs.

export interface RegisterInput {
  email: string;
  password: string;
  displayName: string;
  birthYear?: number;
  ageConfirmed: boolean;
  inviteCode?: string;
}

export const authApi = {
  register(input: RegisterInput) {
    return unwrap<User>(api.post('/auth/register', input));
  },
  registerPartner(input: RegisterInput & { company: string; contactEmail?: string }) {
    return unwrap<{ user: User; partner: PartnerProfile }>(api.post('/auth/register/partner', input));
  },
  login(email: string, password: string, totpCode?: string) {
    return unwrap<{ tokens: Tokens; user: User }>(
      api.post('/auth/login', totpCode ? { email, password, totpCode } : { email, password }),
    );
  },
  logoutAll() {
    return unwrap<{ revoked: boolean }>(api.post('/auth/logout-all'));
  },
  enrollTOTP() {
    return unwrap<{ secret: string; provisioningUri: string }>(api.post('/auth/totp/enroll'));
  },
  confirmTOTPEnrollment(code: string) {
    return unwrap<{ recoveryCodes: string[] }>(api.post('/auth/totp/confirm', { code }));
  },
  disableTOTP(password: string) {
    return unwrap<{ disabled: boolean }>(api.post('/auth/totp/disable', { password }));
  },
  exportData() {
    return unwrap<Record<string, unknown>>(api.get('/auth/export'));
  },
  me() {
    return unwrap<{ userId: string; role: Role; ageConfirmedAt?: string | null; displayName: string; email: string; totpEnabled?: boolean }>(api.get('/auth/me'));
  },
  confirmAge(ageConfirmed: boolean, birthYear?: number) {
    return unwrap<{ confirmed: boolean }>(api.post('/auth/age-confirm', { ageConfirmed, birthYear }));
  },
  withdraw() {
    return unwrap<{ withdrawn: boolean }>(api.post('/auth/withdraw'));
  },
  requestPasswordReset(email: string) {
    return unwrap<{ requested: boolean }>(api.post('/auth/password-reset/request', { email }));
  },
  confirmPasswordReset(token: string, newPassword: string) {
    return unwrap<{ reset: boolean }>(api.post('/auth/password-reset/confirm', { token, newPassword }));
  },
  oauthProviders() {
    return unwrap<OAuthProviderInfo[]>(api.get('/auth/oauth/providers'));
  },
};

export const billingApi = {
  plans() {
    return unwrap<BillingPlan[]>(api.get('/billing/plans'));
  },
  createOrder(planCode: string, refundConsentAt: string) {
    return unwrap<{ order: BillingOrder; plan: BillingPlan }>(
      api.post('/billing/orders', { planCode, refundConsentAt }),
    );
  },
  confirmPayment(paymentKey: string, orderId: string, amount: number) {
    return unwrap<{ subscription: BillingSubscription }>(
      api.post('/billing/payments/confirm', { paymentKey, orderId, amount }),
    );
  },
  mySubscription() {
    return unwrap<{ subscription: BillingSubscription | null }>(api.get('/billing/subscriptions/me'));
  },
  cancelSubscription() {
    return unwrap<{ cancelled: boolean }>(api.post('/billing/subscriptions/cancel'));
  },
  refundActivePass(reason?: string) {
    return unwrap<{ order: BillingOrder }>(api.post('/billing/payments/refund', { reason }));
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
  pendingConsents() {
    return unwrap<LegalTerm[]>(api.get('/legal/consents/pending'));
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
      collectiveTrustStatus?: string;
      collectiveName?: string;
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
  submitGeneration(input: {
    prompt: string;
    lyrics?: string;
    model?: string;
    licenseRef?: string;
    params?: Record<string, unknown>;
  }) {
    return unwrap<GenerationJob>(api.post('/partner/generations', input));
  },
  listGenerations(limit = 20) {
    return unwrap<GenerationJob[]>(api.get('/partner/generations', { params: { limit } }));
  },
  getGeneration(id: string) {
    return unwrap<GenerationJob>(api.get(`/partner/generations/${encodeURIComponent(id)}`));
  },
  registerStore(input: { businessNumber: string; storeName: string; licenseCount?: number }) {
    return unwrap<StoreBusiness>(api.post('/partner/stores', input));
  },
  listStores() {
    return unwrap<StoreBusiness[]>(api.get('/partner/stores'));
  },
  createStoreDevice(storeId: string, deviceName: string) {
    return unwrap<StoreDevice>(api.post(`/partner/stores/${encodeURIComponent(storeId)}/devices`, { deviceName }));
  },
  listStoreDevices(storeId: string) {
    return unwrap<StoreDevice[]>(api.get(`/partner/stores/${encodeURIComponent(storeId)}/devices`));
  },
  updateStoreDevice(storeId: string, deviceId: string, input: { deviceName: string; volume: number }) {
    return unwrap<StoreDevice>(
      api.put(`/partner/stores/${encodeURIComponent(storeId)}/devices/${encodeURIComponent(deviceId)}`, input),
    );
  },
  revokeStoreDevice(storeId: string, deviceId: string) {
    return unwrap<{ revoked: boolean }>(
      api.post(`/partner/stores/${encodeURIComponent(storeId)}/devices/${encodeURIComponent(deviceId)}/revoke`),
    );
  },
  listStoreSchedules(storeId: string) {
    return unwrap<StoreSchedule[]>(api.get(`/partner/stores/${encodeURIComponent(storeId)}/schedules`));
  },
  upsertStoreSchedule(
    storeId: string,
    input: { weekday: number; openTime: string; closeTime: string; playlistId?: string | null; isClosed?: boolean },
  ) {
    return unwrap<StoreSchedule>(api.post(`/partner/stores/${encodeURIComponent(storeId)}/schedules`, input));
  },
  storeDeviceLogs(storeId: string, deviceId: string, limit = 50) {
    return unwrap<StorePlayLog[]>(
      api.get(`/partner/stores/${encodeURIComponent(storeId)}/devices/${encodeURIComponent(deviceId)}/logs`, {
        params: { limit },
      }),
    );
  },
};

export const trackApi = {
  list(
    limit = 20,
    offset = 0,
    filters?: { provider?: string; genre?: string; mood?: string; q?: string; sort?: string },
  ) {
    return unwrap<TrackPage>(api.get('/tracks', { params: { limit, offset, ...filters } }));
  },
  get(id: string) {
    return unwrap<Track>(api.get(`/tracks/${encodeURIComponent(id)}`));
  },
  stream(id: string) {
    return unwrap<StreamGrant>(api.get(`/tracks/${encodeURIComponent(id)}/stream`));
  },
  async download(id: string): Promise<{ blob: Blob; filename: string }> {
    const res = await api.get(`/tracks/${encodeURIComponent(id)}/download`, { responseType: 'blob' });
    const disp = String(res.headers?.['content-disposition'] ?? '');
    const m = /filename="([^"]+)"/.exec(disp);
    return { blob: res.data as Blob, filename: m?.[1] ?? `AVIBE-${id.slice(0, 8)}.bin` };
  },
  getCertificate(id: string) {
    return unwrap<CertificateView>(api.get(`/tracks/${encodeURIComponent(id)}/certificate`));
  },
  verifyCertificate(number: string) {
    return unwrap<CertificateView>(api.get(`/certificates/${encodeURIComponent(number)}`));
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
  overview() {
    return unwrap<AdminOverview>(api.get('/admin/overview'));
  },
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
      maxGeneratedAt?: string | null;
      termsSnapshotUrl?: string | null;
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
  calculateSettlement(
    partnerId: string,
    periodStart: string,
    periodEnd: string,
    storePoolKrw?: number,
    directKrw?: number,
    incomeType?: string,
  ) {
    return unwrap<PartnerSettlement>(
      api.post('/admin/settlements/calculate', { partnerId, periodStart, periodEnd, storePoolKrw, directKrw, incomeType }),
    );
  },
  listSettlements(partnerId: string) {
    return unwrap<PartnerSettlement[]>(api.get(`/admin/settlements/${encodeURIComponent(partnerId)}`));
  },
  approveSettlement(partnerId: string, periodStart: string, periodEnd: string) {
    return unwrap<PartnerSettlement>(api.post('/admin/settlements/approve', { partnerId, periodStart, periodEnd }));
  },
  markSettlementPaid(partnerId: string, periodStart: string, periodEnd: string, reference?: string) {
    return unwrap<PartnerSettlement>(
      api.post('/admin/settlements/mark-paid', { partnerId, periodStart, periodEnd, reference }),
    );
  },
  holdSettlement(partnerId: string, periodStart: string, periodEnd: string, reason: string) {
    return unwrap<PartnerSettlement>(
      api.post('/admin/settlements/hold', { partnerId, periodStart, periodEnd, reason }),
    );
  },
  releaseSettlement(partnerId: string, periodStart: string, periodEnd: string) {
    return unwrap<PartnerSettlement>(
      api.post('/admin/settlements/release', { partnerId, periodStart, periodEnd }),
    );
  },
  issueTrackCertificate(trackId: string) {
    return unwrap<TrackCertificate>(
      api.post(`/admin/tracks/${encodeURIComponent(trackId)}/certificate`, {}),
    );
  },
  getPartnerSettlementAccount(partnerId: string) {
    return unwrap<{ bank: string; accountMasked: string }>(
      api.get(`/admin/partners/${encodeURIComponent(partnerId)}/settlement`),
    );
  },
  listUserBlocks(id: string) {
    return unwrap<UserBlock[]>(api.get(`/admin/users/${encodeURIComponent(id)}/blocks`));
  },
  auditEvents(params: { action?: string; entityType?: string; limit?: number; offset?: number } = {}) {
    return unwrap<AuditEvent[]>(api.get('/admin/audit-events', { params }));
  },
  listUsers(params: { q?: string; role?: string; limit?: number; offset?: number } = {}) {
    return unwrap<User[]>(api.get('/admin/users', { params }));
  },
  setUserRole(id: string, role: string) {
    return unwrap<User>(api.post(`/admin/users/${encodeURIComponent(id)}/role`, { role }));
  },
  setUserActive(id: string, active: boolean) {
    return unwrap<User>(api.post(`/admin/users/${encodeURIComponent(id)}/active`, { active }));
  },
  setTrackISRC(id: string, isrc: string) {
    return unwrap<Track>(api.put(`/admin/tracks/${encodeURIComponent(id)}/isrc`, { isrc }));
  },
  listStores(status = 'PENDING', limit = 20, offset = 0) {
    return unwrap<StoreBusiness[]>(api.get('/admin/stores', { params: { status, limit, offset } }));
  },
  verifyStore(id: string, approve: boolean) {
    return unwrap<StoreBusiness>(api.post(`/admin/stores/${encodeURIComponent(id)}/verify`, { approve }));
  },
  dspKit(trackIds: string[]) {
    return unwrap<DSPKit>(api.get('/admin/distribution/dsp-kit', { params: { trackIds: trackIds.join(',') } }));
  },
  async dspKitCsv(trackIds: string[]): Promise<Blob> {
    const res = await api.get('/admin/distribution/dsp-kit', {
      params: { trackIds: trackIds.join(','), format: 'csv' },
      responseType: 'blob',
    });
    return res.data as Blob;
  },
};

export const storeDeviceApi = {
  claim(pairingCode: string) {
    return unwrap<{ device: StoreDevice; token: string }>(
      deviceApiClient.post('/store/device/claim', { pairingCode }),
    );
  },
  me() {
    return unwrap<StoreDevice>(deviceApiClient.get('/store/device/me'));
  },
  queue() {
    return unwrap<DeviceQueue>(deviceApiClient.get('/store/device/queue'));
  },
  heartbeat(trackId: string, durationSecs: number) {
    return unwrap<{ counted: boolean }>(
      deviceApiClient.post('/store/device/heartbeat', { trackId, durationSecs }),
    );
  },
  stream(trackId: string) {
    return unwrap<StreamGrant>(deviceApiClient.get(`/store/device/stream/${encodeURIComponent(trackId)}`));
  },
};
