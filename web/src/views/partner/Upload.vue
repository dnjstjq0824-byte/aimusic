<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <!-- Header -->
    <div class="border-b border-surface-border pb-4">
      <div class="flex items-center gap-2 text-xs text-brand-400 font-bold uppercase tracking-wider mb-1">
        <span>Gatekeeper Pipeline</span>
      </div>
      <h1 class="text-2xl font-black text-white tracking-tight">AI 음원 업로드 &amp; 상업 라이선스 검증 위저드</h1>
      <p class="text-xs text-neutral-400 mt-1">
        상업용 라이선스가 보증된 AI 음원만 등록 가능합니다. 비상업용 및 소급 불가 음원은 원천 차단됩니다.
      </p>
    </div>

    <!-- Stepper indicator -->
    <div class="grid grid-cols-3 gap-2">
      <div
        v-for="(st, idx) in steps"
        :key="st.num"
        class="flex items-center gap-2.5 rounded-xl border p-3 transition-all text-xs"
        :class="store.step === st.num ? 'border-brand-500 bg-brand-950/40 text-brand-300 font-bold shadow-glow-brand' : store.step > st.num ? 'border-emerald-600/40 bg-surface-1 text-emerald-400 font-semibold' : 'border-surface-border bg-surface-1 text-neutral-500'"
      >
        <div
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px] font-bold"
          :class="store.step === st.num ? 'bg-brand-500 text-white' : store.step > st.num ? 'bg-emerald-600 text-white' : 'bg-surface-2 text-neutral-400'"
        >
          {{ store.step > st.num ? '✓' : idx + 1 }}
        </div>
        <div class="truncate">
          <div class="font-bold">{{ st.title }}</div>
          <div class="text-[10px] opacity-70 hidden sm:block">{{ st.desc }}</div>
        </div>
      </div>
    </div>

    <!-- Error Banner -->
    <div v-if="loadError" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ loadError }}
    </div>

    <!-- Step 1: Pre-check Gate 1 & 2 -->
    <UiCard v-if="store.step === 1" class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">Step 1 — 상업 라이선스 사전 검증 (Gate 1 &amp; Gate 2)</h2>
        <p class="text-xs text-neutral-400 mt-0.5">
          제작에 사용한 AI 플랫폼과 플랜, 음원 생성 일시를 검증하여 단독 상업 유통 자격을 확인합니다.
        </p>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <UiSelect
          v-model="store.provider"
          label="AI 작곡 플랫폼"
          required
          @change="store.planTierUsed = ''"
        >
          <option value="">플랫폼을 선택하세요…</option>
          <option v-for="p in store.policies" :key="p.provider" :value="p.provider">
            {{ p.provider }} {{ p.allowsStandaloneStreaming ? '(상업 유통 허용)' : '(Sync-only 영상BGM 전용 - 단독 유통 불가)' }}
          </option>
        </UiSelect>

        <UiSelect
          v-model="store.planTierUsed"
          label="생성 당시 사용 요금제 (Plan Tier)"
          required
        >
          <option value="">요금제를 선택하세요…</option>
          <option v-for="tier in store.availablePlanTiers" :key="tier" :value="tier">
            {{ store.provider }} — {{ tier }}
          </option>
        </UiSelect>

        <UiSelect
          v-model="store.subscriptionId"
          label="등록된 유료 구독 영수증 증빙"
          required
        >
          <option value="">구독 증빙을 선택하세요…</option>
          <option v-for="s in store.subscriptions" :key="s.id" :value="s.id">
            {{ s.provider }} / {{ s.planName }} ({{ s.status }})
          </option>
        </UiSelect>

        <UiInput
          v-model="store.generatedAt"
          type="datetime-local"
          label="음원 최초 생성일시 (소급 방지 대조)"
          required
        />

        <div class="sm:col-span-2">
          <UiInput
            v-model="store.providerTrackUrl"
            label="AI 플랫폼 원본 트랙 공유 URL (선택 검증용)"
            placeholder="https://suno.com/song/... 또는 https://www.udio.com/songs/..."
          />
        </div>

        <div class="sm:col-span-2">
          <label class="block text-xs font-medium text-neutral-300 mb-1.5">
            오디오 파일 선택 (무손실 WAV, FLAC, 또는 192k+ MP3)
          </label>
          <input
            type="file"
            accept="audio/*"
            :disabled="store.instantBlockReason !== null"
            class="w-full text-xs text-neutral-400 file:mr-3 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-surface-2 file:text-white hover:file:bg-surface-3 cursor-pointer"
            @change="onFile"
          />
        </div>
      </div>

      <!-- Real-time Instant Block Warning Banner -->
      <div
        v-if="store.instantBlockReason"
        class="rounded-xl border border-red-600 bg-red-950/70 p-4 text-xs font-semibold text-red-200 flex items-start gap-2.5 shadow-lg"
      >
        <span class="text-base">⛔</span>
        <div class="leading-relaxed">
          {{ store.instantBlockReason }}
        </div>
      </div>

      <LicensePrecheckWidget :status="store.gate12" :code="store.gate12Code" />

      <div class="flex items-center justify-between pt-2 border-t border-surface-border">
        <p class="text-xs text-neutral-500">
          * 사전 검증을 통과해야 실제 파일 업로드 및 Step 2가 해제됩니다.
        </p>
        <div class="flex gap-2">
          <UiButton
            variant="brand"
            size="sm"
            :loading="store.gate12 === 'checking'"
            :disabled="store.instantBlockReason !== null"
            @click="precheck"
          >
            Gate 1-2 검증 실행
          </UiButton>
          <UiButton
            variant="secondary"
            size="sm"
            :disabled="store.step2Blocked"
            @click="store.step = 2"
          >
            Step 2로 진행 →
          </UiButton>
        </div>
      </div>
    </UiCard>

    <!-- Step 2: File Upload & Forensic (Gate 3) -->
    <UiCard v-else-if="store.step === 2" class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">Step 2 — 오디오 업로드 &amp; FFprobe 포렌식 (Gate 3)</h2>
        <p class="text-xs text-neutral-400 mt-0.5">
          파일의 무결성(SHA-256)을 확인하고, 오디오 스트림 메타데이터 내 무료 워터마크 태그 및 음질을 분석합니다.
        </p>
      </div>

      <div class="rounded-xl border border-surface-border bg-surface-0 p-4 space-y-2 text-xs">
        <div class="flex justify-between">
          <span class="text-neutral-400">선택된 파일:</span>
          <span class="font-mono text-white font-semibold">{{ store.fileName || '선택 안됨' }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-neutral-400">파일 고유 SHA-256:</span>
          <span class="font-mono text-brand-400 break-all text-[11px]">{{ store.audioSha256 || '계산 대기' }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-neutral-400">Gate 1-2 사전 승인:</span>
          <span class="text-accent-emerald font-semibold">✓ 통과 (ELIGIBLE)</span>
        </div>
      </div>

      <div v-if="uploadError" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
        {{ uploadError }}
      </div>

      <div class="flex items-center justify-between pt-2 border-t border-surface-border">
        <UiButton variant="ghost" size="sm" @click="store.step = 1">← 이전 단계</UiButton>
        <UiButton
          variant="brand"
          size="sm"
          :loading="uploading"
          @click="uploadAndVerify"
        >
          오디오 스테이징 업로드 및 포렌식 통과 →
        </UiButton>
      </div>
    </UiCard>

    <!-- Step 3: AI Metadata & 4 Legal Warranties -->
    <UiCard v-else class="space-y-5">
      <div>
        <h2 class="text-sm font-bold text-white">Step 3 — AI 메타데이터 &amp; 4대 필수 법적 서약</h2>
        <p class="text-xs text-neutral-400 mt-0.5">
          인공지능기본법 제31조 표시 의무와 저작권 면책을 위한 법적 확약서를 작성합니다.
        </p>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <UiInput
          v-model="store.title"
          label="음원 제목 (Title)"
          placeholder="예: Cyber Neon Horizon"
          required
        />
        <UiInput
          v-model="store.genre"
          label="음악 장르"
          placeholder="Synthwave / Lo-Fi / Cinematic"
        />
        <UiInput
          v-model="store.moodTagsInput"
          label="무드 / 분위기 태그 (쉼표로 구분)"
          placeholder="Focus, Cyberpunk, Night"
        />
        <UiSelect
          v-model="store.disclosure"
          label="AI 생성 유형 (투명성 고지)"
        >
          <option value="FULL_AI">순수 AI 생성물 (Full AI)</option>
          <option value="AI_ASSISTED_HUMAN">AI + 인간 공동 창작 (Human-in-the-loop)</option>
        </UiSelect>

        <div class="sm:col-span-2">
          <UiInput
            v-model="store.aiPrompt"
            label="사용된 생성 프롬프트 (선택 공개)"
            placeholder="예: cinematic synthwave retro arpeggio 120bpm"
          />
        </div>

        <div class="sm:col-span-2">
          <UiInput
            v-model="store.humanContributionDesc"
            label="인간 창작 기여 내역 설명 (해당 시)"
            placeholder="직접 작사, 마스터링 EQ 조정, 보컬 멜로디 가이드 등"
          />
        </div>
      </div>

      <!-- 4 Mandatory Legal Warranties -->
      <div class="space-y-3 rounded-2xl border border-brand-500/30 bg-surface-0 p-4">
        <div class="text-xs font-bold text-white uppercase tracking-wider mb-2 flex items-center gap-1.5">
          <span>⚖️ 4대 필수 법적 보증 서약서</span>
          <span class="text-accent-crimson">*</span>
        </div>

        <UiCheckbox v-model="store.declaresOwnership" required>
          <strong>[상업용 라이선스 보유 확약]</strong> 본 음원은 단독 스트리밍 유통이 명시된 유료 상업용 플랜 유효 기간 내에 생성되었음을 보증합니다.
        </UiCheckbox>

        <UiCheckbox v-model="store.declaresTrainingConsent" required>
          <strong>[무료 생성 소급 음원 부존재 확약]</strong> 무료 플랜 사용 중 생성한 뒤 사후에 유료 전환한 소급 적용 불가 음원이 아님을 보증합니다.
        </UiCheckbox>

        <UiCheckbox v-model="store.declaresNoVoiceClone" required>
          <strong>[보이스 클로닝 및 무단 샘플링 부존재 확약]</strong> 타인의 음성(퍼블리시티권)을 무단 복제하거나 기성 저작물을 표절하지 않았음을 보증합니다.
        </UiCheckbox>

        <UiCheckbox v-model="store.agreesCommercialTerms" required>
          <strong>[면책 및 허위 등록 시 제재 동의]</strong> 허위 진술 시 플랫폼에 대한 민·형사상 면책 및 발생 수익 전액 몰수 조치에 동의합니다.
        </UiCheckbox>
      </div>

      <div class="flex items-center justify-between pt-2 border-t border-surface-border">
        <UiButton variant="ghost" size="sm" @click="store.step = 2">← 이전 단계</UiButton>
        <UiButton
          variant="brand"
          size="sm"
          :disabled="!store.canFinalize"
          :loading="store.finalizing"
          @click="finalizeUpload"
        >
          ✓ 최종 제출 및 어드민 검수 요청
        </UiButton>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { usePartnerUploadStore } from '@/stores/partnerUploadStore';
import { useToast } from '@/composables/useToast';
import { errMessageOf } from '@/api/client';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import LicensePrecheckWidget from '@/components/legal/LicensePrecheckWidget.vue';

const store = usePartnerUploadStore();
const router = useRouter();
const toast = useToast();

const steps = [
  { num: 1, title: 'Gate 1-2 라이선스', desc: '플랫폼 및 생성일자' },
  { num: 2, title: 'Gate 3 오디오 분석', desc: 'SHA-256 및 포렌식' },
  { num: 3, title: 'AI 공시 & 4대 서약', desc: '메타데이터 및 최종 제출' },
];

const loadError = ref<string | null>(null);
const uploadError = ref<string | null>(null);
const uploading = ref(false);

function onFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  store.pickFile(file);
}

async function precheck() {
  uploadError.value = null;
  const ok = await store.runPrecheck();
  if (ok) {
    toast.success('Gate 1-2 상업 라이선스 검증을 통과했습니다!');
  } else {
    toast.error(store.gate12Message || 'Gate 1-2 검증에 통과하지 못했습니다.');
  }
}

async function uploadAndVerify() {
  uploading.value = true;
  uploadError.value = null;
  try {
    const ok = await store.uploadStaged();
    if (ok) {
      toast.success('오디오 파일 업로드 및 포렌식 대조 완료!');
      store.step = 3;
    } else {
      uploadError.value = store.gate3Message || '업로드에 실패했습니다.';
    }
  } catch (err) {
    uploadError.value = errMessageOf(err);
  } finally {
    uploading.value = false;
  }
}

async function finalizeUpload() {
  const ok = await store.finalize();
  if (ok) {
    toast.success('음원 등록이 완료되었습니다. 관리자 승인 후 대외 서비스됩니다.');
    store.reset();
    await router.push('/partner');
  } else {
    toast.error(store.finalizeError || '음원 등록에 실패했습니다.');
  }
}

onMounted(async () => {
  try {
    await store.loadReference();
  } catch (err) {
    loadError.value = errMessageOf(err);
  }
});
</script>
