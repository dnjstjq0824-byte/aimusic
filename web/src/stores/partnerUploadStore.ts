import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errCodeOf } from '@/api/client';
import type { AIDisclosure, AIProviderPolicy, PartnerAISubscription, PrecheckOutput } from '@/api/types';

export type GateStatus = 'idle' | 'checking' | 'pass' | 'fail';

// Gate1-2 = provider commercial eligibility + generation-date/subscription checks
// (server-side licensegate). Gate3 = file-level checks at finalize/transcode.
export const usePartnerUploadStore = defineStore('partnerUpload', () => {
  const step = ref(1);
  const provider = ref('');
  const planTierUsed = ref('');
  const providerTrackUrl = ref('');
  const subscriptionId = ref('');
  const generatedAt = ref('');
  const fileName = ref('');
  const audioSha256 = ref('');
  const file = ref<File | null>(null);

  const gate12 = ref<GateStatus>('idle');
  const gate12Code = ref<string | null>(null);
  const gate12Message = ref<string | null>(null);
  const grant = ref<PrecheckOutput | null>(null);

  const gate3 = ref<GateStatus>('idle');
  const gate3Message = ref<string | null>(null);
  const uploading = ref(false);
  const uploadProgress = ref(0);
  const stagedKey = ref<string | null>(null);

  const title = ref('');
  const albumId = ref('');
  const genre = ref('Electronic');
  const moodTagsInput = ref('Chill, Cyberpunk');
  const disclosure = ref<AIDisclosure>('FULL_AI');
  const sourceModel = ref('');
  const aiPrompt = ref('');
  const isPromptPublic = ref(true);
  const isInstrumental = ref(false);
  const lyrics = ref('');
  const humanContributionDesc = ref('');
  const collectiveTrust = ref('NONE');
  const collectiveName = ref('');
  const declaresOwnership = ref(false);
  const declaresNoVoiceClone = ref(false);
  const declaresTrainingConsent = ref(false);
  const agreesCommercialTerms = ref(false);
  const finalized = ref<{ trackId: string; jobId: string } | null>(null);
  const finalizing = ref(false);
  const finalizeError = ref<string | null>(null);

  const policies = ref<AIProviderPolicy[]>([]);
  const subscriptions = ref<PartnerAISubscription[]>([]);

  const selectedPolicy = computed(() => policies.value.find((p) => p.provider === provider.value) ?? null);
  const selectedSubscription = computed(() => subscriptions.value.find((s) => s.id === subscriptionId.value) ?? null);

  const availablePlanTiers = computed(() => {
    const p = selectedPolicy.value;
    if (!p) return [];
    return [
      ...(p.allowedPlanTiers ?? []),
      ...(p.syncOnlyPlanTiers ?? []),
      ...(p.blockedPlanTiers ?? []),
    ];
  });

  // Real-time client-side Gate 1 & Gate 2 instant block banner reason
  const instantBlockReason = computed<string | null>(() => {
    const p = selectedPolicy.value;
    if (!p) return null;
    if (!p.allowsStandaloneStreaming) {
      return `선택하신 요금제 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다. (${p.rejectionMessageKo})`;
    }
    const tier = (planTierUsed.value || selectedSubscription.value?.planName || '').trim().toLowerCase();
    if (tier) {
      const isBlocked = (p.blockedPlanTiers ?? []).some((t) => t.toLowerCase() === tier);
      if (isBlocked) {
        return `선택하신 요금제(${planTierUsed.value || selectedSubscription.value?.planName}) 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다. (비상업/무료 요금제 차단: ERR_NON_COMMERCIAL_TIER)`;
      }
      const isSyncOnly = (p.syncOnlyPlanTiers ?? []).some((t) => t.toLowerCase() === tier);
      if (isSyncOnly) {
        return `선택하신 요금제(${planTierUsed.value || selectedSubscription.value?.planName}) 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다. (영상 배경음악 Sync 전용 라이선스 차단: ERR_SYNC_ONLY_LICENSE)`;
      }
    }
    if (generatedAt.value && selectedSubscription.value) {
      const genTime = new Date(generatedAt.value).getTime();
      const subStart = new Date(selectedSubscription.value.createdAt).getTime();
      if (!Number.isNaN(genTime) && !Number.isNaN(subStart) && genTime < subStart) {
        return '선택하신 요금제 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다. (유료 구독 시작일 이전에 생성된 음원 소급 적용 불가: ERR_RETROACTIVE_GENERATION_DATE)';
      }
      if (selectedSubscription.value.expiresAt) {
        const subEnd = new Date(selectedSubscription.value.expiresAt).getTime();
        if (!Number.isNaN(genTime) && !Number.isNaN(subEnd) && genTime > subEnd) {
          return '선택하신 요금제 또는 생성 시점의 음원은 상업적 단독 음원 스트리밍 유통이 불가하여 업로드할 수 없습니다. (유료 구독 만료 이후 생성된 음원: ERR_NON_COMMERCIAL_TIER)';
        }
      }
    }
    return null;
  });

  const step2Blocked = computed(() => instantBlockReason.value !== null || gate12.value !== 'pass' || !grant.value);
  const canFinalize = computed(
    () =>
      stagedKey.value !== null &&
      title.value.trim().length > 0 &&
      collectiveTrust.value === 'NONE' &&
      declaresOwnership.value &&
      declaresNoVoiceClone.value &&
      declaresTrainingConsent.value &&
      agreesCommercialTerms.value &&
      !finalizing.value,
  );

  async function loadReference(): Promise<void> {
    const [p, s] = await Promise.all([partnerApi.listPolicies(), partnerApi.listSubscriptions()]);
    policies.value = p;
    subscriptions.value = s;
  }

  async function sha256Of(f: File): Promise<string> {
    const buf = await f.arrayBuffer();
    const digest = await crypto.subtle.digest('SHA-256', buf);
    return [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, '0')).join('');
  }

  function pickFile(f: File): void {
    file.value = f;
    fileName.value = f.name;
    gate12.value = 'idle';
    gate12Code.value = null;
    gate12Message.value = null;
    grant.value = null;
  }

  // Step1: Gate1-2 pre-check. On fail, block banner disables Step2.
  async function runPrecheck(): Promise<boolean> {
    if (instantBlockReason.value) {
      gate12.value = 'fail';
      gate12Code.value = 'ERR_NON_COMMERCIAL_TIER';
      gate12Message.value = instantBlockReason.value;
      return false;
    }
    if (!file.value) {
      gate12.value = 'fail';
      gate12Code.value = 'ERR_INVALID_INPUT';
      gate12Message.value = '음원 파일을 먼저 선택해 주세요.';
      return false;
    }
    gate12.value = 'checking';
    gate12Code.value = null;
    gate12Message.value = null;
    grant.value = null;
    try {
      const sha = await sha256Of(file.value);
      audioSha256.value = sha;
      const out = await partnerApi.preCheck({
        provider: provider.value,
        planTierUsed: planTierUsed.value || selectedSubscription.value?.planName || undefined,
        audioSha256: sha,
        fileName: fileName.value,
        generatedAt: new Date(generatedAt.value).toISOString(),
        subscriptionId: subscriptionId.value || undefined,
        requiresActiveSub: provider.value !== 'CUSTOM_OPEN_MODEL',
      });
      grant.value = out;
      gate12.value = 'pass';
      return true;
    } catch (err: unknown) {
      gate12.value = 'fail';
      gate12Code.value = errCodeOf(err);
      gate12Message.value = gate12Code.value;
      return false;
    }
  }

  // Step2: PUT file to presigned staging URL, then Gate3 local status.
  async function uploadStaged(): Promise<boolean> {
    if (!file.value || !grant.value) return false;
    uploading.value = true;
    uploadProgress.value = 0;
    gate3.value = 'checking';
    gate3Message.value = null;
    try {
      await new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('PUT', grant.value?.uploadUrl ?? '');
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) uploadProgress.value = Math.round((e.loaded / e.total) * 100);
        };
        xhr.onload = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`upload:${xhr.status}`)));
        xhr.onerror = () => reject(new Error('upload:network'));
        const f = file.value;
        if (!f) {
          reject(new Error('upload:nofile'));
          return;
        }
        xhr.send(f);
      });
      stagedKey.value = grant.value.stagingKey;
      uploadProgress.value = 100;
      gate3.value = 'pass';
      gate3Message.value = '스테이징 업로드 완료. 파일 무결성/형식 검사가 finalize 단계에서 진행됩니다.';
      return true;
    } catch (err: unknown) {
      gate3.value = 'fail';
      gate3Message.value = err instanceof Error ? err.message : 'upload failed';
      return false;
    } finally {
      uploading.value = false;
    }
  }

  function disclosureText(): string {
    return disclosure.value === 'FULL_AI'
      ? 'FULL_AI: 본 음원은 AI가 전곡 생성한 음원입니다.'
      : 'AI_ASSISTED_HUMAN: 본 음원은 인간 창작자가 AI 보조 도구를 사용해 제작한 음원입니다.';
  }

  // Step3: metadata + rights finalize.
  async function finalize(bitrateKbps?: number, durationMs?: number): Promise<boolean> {
    if (!canFinalize.value || !stagedKey.value) return false;
    finalizing.value = true;
    finalizeError.value = null;
    try {
      const moodTags = moodTagsInput.value
        .split(',')
        .map((m) => m.trim())
        .filter(Boolean);
      const effectiveTier = planTierUsed.value || selectedSubscription.value?.planName || undefined;
      const out = await partnerApi.finalize({
        stagingKey: stagedKey.value,
        title: title.value.trim(),
        provider: provider.value,
        planTierUsed: effectiveTier,
        audioSha256: audioSha256.value,
        generatedAt: new Date(generatedAt.value).toISOString(),
        subscriptionId: subscriptionId.value || undefined,
        albumId: albumId.value || undefined,
        genre: genre.value.trim() || undefined,
        moodTags,
        aiGenerationType: disclosure.value,
        aiModelVersion: sourceModel.value.trim() || undefined,
        aiPrompt: aiPrompt.value.trim() || undefined,
        isPromptPublic: isPromptPublic.value,
        isInstrumental: isInstrumental.value,
        lyrics: lyrics.value.trim() || undefined,
        durationMs,
        bitrateKbps,
        disclosure: disclosureText(),
        rights: {
          declaresOwnership: declaresOwnership.value,
          declaresNoVoiceClone: declaresNoVoiceClone.value,
          declaresTrainingConsent: declaresTrainingConsent.value,
          confirmedNoUnauthorizedSampling: declaresTrainingConsent.value,
          confirmedIndemnification: agreesCommercialTerms.value,
          planTierUsed: effectiveTier,
          providerTrackUrl: providerTrackUrl.value.trim() || undefined,
          humanContributionDesc: humanContributionDesc.value.trim() || undefined,
          sourceModel: sourceModel.value || undefined,
          subscriptionId: subscriptionId.value || undefined,
          collectiveTrustStatus: collectiveTrust.value,
          collectiveName: collectiveName.value.trim() || undefined,
        },
      });
      finalized.value = out;
      return true;
    } catch (err: unknown) {
      finalizeError.value = errCodeOf(err);
      return false;
    } finally {
      finalizing.value = false;
    }
  }

  function reset(): void {
    step.value = 1;
    provider.value = '';
    planTierUsed.value = '';
    providerTrackUrl.value = '';
    subscriptionId.value = '';
    generatedAt.value = '';
    fileName.value = '';
    audioSha256.value = '';
    file.value = null;
    gate12.value = 'idle';
    gate12Code.value = null;
    gate12Message.value = null;
    grant.value = null;
    gate3.value = 'idle';
    gate3Message.value = null;
    uploading.value = false;
    uploadProgress.value = 0;
    stagedKey.value = null;
    title.value = '';
    albumId.value = '';
    genre.value = 'Electronic';
    moodTagsInput.value = 'Chill, Cyberpunk';
    disclosure.value = 'FULL_AI';
    sourceModel.value = '';
    aiPrompt.value = '';
    isPromptPublic.value = true;
    isInstrumental.value = false;
    lyrics.value = '';
    humanContributionDesc.value = '';
    collectiveTrust.value = 'NONE';
    collectiveName.value = '';
    declaresOwnership.value = false;
    declaresNoVoiceClone.value = false;
    declaresTrainingConsent.value = false;
    agreesCommercialTerms.value = false;
    finalized.value = null;
    finalizing.value = false;
    finalizeError.value = null;
  }

  return {
    step, provider, planTierUsed, providerTrackUrl, subscriptionId, generatedAt, fileName, audioSha256, file,
    gate12, gate12Code, gate12Message, grant,
    gate3, gate3Message, uploading, uploadProgress, stagedKey,
    title, albumId, genre, moodTagsInput, disclosure, sourceModel,
    aiPrompt, isPromptPublic, isInstrumental, lyrics, humanContributionDesc,
    collectiveTrust, collectiveName,
    declaresOwnership, declaresNoVoiceClone, declaresTrainingConsent, agreesCommercialTerms,
    finalized, finalizing, finalizeError,
    policies, subscriptions, selectedPolicy, availablePlanTiers, instantBlockReason, step2Blocked, canFinalize,
    loadReference, pickFile, runPrecheck, uploadStaged, finalize, reset,
  };
});
