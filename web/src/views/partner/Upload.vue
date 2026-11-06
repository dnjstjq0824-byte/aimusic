<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Upload Wizard</h1>
    <ol class="mb-4 flex gap-2 text-sm">
      <li v-for="n in [1, 2, 3]" :key="n" class="rounded px-3 py-1" :class="store.step === n ? 'bg-brand-600 font-bold' : 'bg-neutral-800'">
        Step {{ n }}
      </li>
    </ol>
    <p v-if="loadError" class="mb-2 text-sm text-red-400">{{ loadError }}</p>

    <!-- Step 1: Gate1-2 pre-check -->
    <section v-if="store.step === 1" class="space-y-3 rounded bg-neutral-900 p-4">
      <h2 class="font-bold">Step 1 — 상업 라이선스 사전 검증 (Gate 1 &amp; Gate 2)</h2>
      <div class="grid gap-2 sm:grid-cols-2">
        <label class="text-sm">AI 제공자
          <select v-model="store.provider" class="mt-1 w-full rounded bg-neutral-800 p-2" @change="store.planTierUsed = ''">
            <option value="">선택…</option>
            <option v-for="p in store.policies" :key="p.provider" :value="p.provider">
              {{ p.provider }} {{ p.allowsStandaloneStreaming ? '' : '(Sync-only 단독 스트리밍 불가)' }}
            </option>
          </select>
        </label>
        <label class="text-sm">생성 당시 사용 요금제 (Plan Tier)
          <select v-model="store.planTierUsed" class="mt-1 w-full rounded bg-neutral-800 p-2">
            <option value="">구독 요금제 자동 적용 / 직접 선택…</option>
            <option v-for="tier in store.availablePlanTiers" :key="tier" :value="tier">
              {{ store.provider }} - {{ tier }}
            </option>
          </select>
        </label>
        <label class="text-sm">등록된 유료 구독 증빙
          <select v-model="store.subscriptionId" class="mt-1 w-full rounded bg-neutral-800 p-2">
            <option value="">선택…</option>
            <option v-for="s in store.subscriptions" :key="s.id" :value="s.id">
              {{ s.provider }} / {{ s.planName }} ({{ s.status }})
            </option>
          </select>
        </label>
        <label class="text-sm">음원 생성일시
          <input v-model="store.generatedAt" type="datetime-local" class="mt-1 w-full rounded bg-neutral-800 p-2" />
        </label>
        <label class="text-sm sm:col-span-2">AI 플랫폼 원본 트랙 공유 URL (선택)
          <input
            v-model="store.providerTrackUrl"
            placeholder="https://suno.com/song/... 또는 https://www.udio.com/songs/..."
            class="mt-1 w-full rounded bg-neutral-800 p-2"
          />
        </label>
        <label class="text-sm sm:col-span-2">음원 파일 (무손실 WAV/FLAC 또는 192kbps+ 고음질 오디오)
          <input
            type="file"
            accept="audio/*"
            :disabled="store.instantBlockReason !== null"
            class="mt-1 w-full text-sm disabled:cursor-not-allowed disabled:opacity-50"
            @change="onFile"
          />
        </label>
      </div>

      <!-- Real-time Instant Block Warning Banner -->
      <div
        v-if="store.instantBlockReason"
        class="rounded border border-red-600 bg-red-950/60 p-3 text-sm font-semibold text-red-200"
      >
        ⚠️ {{ store.instantBlockReason }}
      </div>

      <LicensePrecheckWidget :status="store.gate12" :code="store.gate12Code" />
      <div class="flex gap-2">
        <button
          class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50"
          :disabled="store.gate12 === 'checking' || store.instantBlockReason !== null"
          @click="precheck"
        >
          {{ store.gate12 === 'checking' ? '검사 중…' : 'Gate1-2 검사 실행' }}
        </button>
        <button
          class="rounded bg-neutral-700 px-3 py-1.5 text-sm font-bold disabled:opacity-50"
          :disabled="store.step2Blocked"
          title="Gate1-2 통과 후 활성화됩니다"
          @click="store.step = 2"
        >
          Step 2로 이동
        </button>
      </div>
      <p v-if="store.gate12 === 'fail' || store.instantBlockReason" class="text-xs text-red-400">
        검사를 통과하지 못하면 파일 업로드 및 Step 2가 비활성화됩니다. 상업용 구독·생성일·제공자 정책을 확인해 주세요.
      </p>
    </section>

    <!-- Step 2: file upload + Gate3 -->
    <section v-if="store.step === 2" class="space-y-3 rounded bg-neutral-900 p-4">
      <h2 class="font-bold">Step 2 — 파일 업로드 (Gate3 상태)</h2>
      <div v-if="store.step2Blocked" class="rounded border border-red-700 bg-red-900/30 p-3 text-sm text-red-200">
        Step 1 Gate1-2를 먼저 통과해야 합니다.
        <button class="ml-2 underline" @click="store.step = 1">Step 1로 돌아가기</button>
      </div>
      <div v-else class="space-y-2 text-sm">
        <p>파일: <span class="font-mono">{{ store.fileName }}</span></p>
        <p>SHA-256: <span class="break-all font-mono text-xs">{{ store.audioSha256 }}</span></p>
        <p>Staging key: <span class="break-all font-mono text-xs">{{ store.grant?.stagingKey }}</span></p>
        <div class="h-2 rounded bg-neutral-800">
          <div class="h-2 rounded bg-brand-500" :style="{ width: `${store.uploadProgress}%` }"></div>
        </div>
        <p v-if="store.gate3Message" class="text-xs text-neutral-300">Gate3: {{ store.gate3Message }}</p>
        <div class="flex gap-2">
          <button class="rounded bg-neutral-700 px-3 py-1.5 text-sm" @click="store.step = 1">이전</button>
          <button
            class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50"
            :disabled="store.uploading || store.stagedKey !== null"
            @click="upload"
          >
            {{ store.uploading ? `업로드 중 ${store.uploadProgress}%…` : store.stagedKey ? '업로드 완료' : '스테이징 업로드' }}
          </button>
          <button
            class="rounded bg-neutral-700 px-3 py-1.5 text-sm font-bold disabled:opacity-50"
            :disabled="store.stagedKey === null"
            @click="store.step = 3"
          >
            Step 3로 이동
          </button>
        </div>
      </div>
    </section>

    <!-- Step 3: metadata + disclosure + legal checkboxes -->
    <section v-if="store.step === 3" class="space-y-3 rounded bg-neutral-900 p-4">
      <h2 class="font-bold">Step 3 — 메타데이터 및 권리 선언</h2>
      <div v-if="!store.stagedKey" class="rounded border border-red-700 bg-red-900/30 p-3 text-sm text-red-200">
        Step 2 업로드를 먼저 완료해야 합니다.
        <button class="ml-2 underline" @click="store.step = 2">Step 2로 돌아가기</button>
      </div>
      <div v-else class="space-y-2">
        <label class="block text-sm">트랙 제목
          <input v-model="store.title" class="mt-1 w-full rounded bg-neutral-800 p-2" />
        </label>
        <div class="grid gap-2 sm:grid-cols-2">
          <label class="block text-sm">앨범
            <select v-model="store.albumId" class="mt-1 w-full rounded bg-neutral-800 p-2">
              <option value="">(없음)</option>
              <option v-for="a in albums" :key="a.id" :value="a.id">{{ a.title }}</option>
            </select>
          </label>
          <label class="block text-sm">소스 모델 버전
            <input v-model="store.sourceModel" placeholder="예: Suno v4 / Udio v1.5" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
          <label class="block text-sm">장르 (Genre)
            <input v-model="store.genre" placeholder="예: Electronic, Ambient, Synthwave" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
          <label class="block text-sm">무드 태그 (쉼표 구분)
            <input v-model="store.moodTagsInput" placeholder="예: Chill, Focus, Cyberpunk" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
        </div>
        <fieldset class="text-sm">
          <legend class="font-semibold">AI 공개 유형</legend>
          <label class="mr-4"><input v-model="store.disclosure" type="radio" value="FULL_AI" /> FULL_AI (100% AI 생성)</label>
          <label><input v-model="store.disclosure" type="radio" value="AI_ASSISTED_HUMAN" /> AI_ASSISTED_HUMAN (인간+AI 공동 창작)</label>
        </fieldset>
        <div v-if="store.disclosure === 'AI_ASSISTED_HUMAN'" class="text-sm">
          <label class="block">인간 창작 기여 내역 (한국저작권위원회 AI-인간 공동창작물 등록 대비)
            <input
              v-model="store.humanContributionDesc"
              placeholder="예: 직접 작사 100%, 후처리 믹싱 및 보컬 멜로디 편곡 수행"
              class="mt-1 w-full rounded bg-neutral-800 p-2"
            />
          </label>
        </div>
        <div class="grid gap-2 sm:grid-cols-2">
          <label class="block text-sm">AI 생성 프롬프트 (선택)
            <textarea v-model="store.aiPrompt" rows="2" placeholder="생성 시 사용한 프롬프트" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
          <label class="block text-sm">가사 (Lyrics / 연주곡은 비워둠)
            <textarea v-model="store.lyrics" rows="2" placeholder="트랙 가사 입력" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
        </div>
        <fieldset class="space-y-1 text-sm">
          <legend class="font-semibold">법적 보증 서약 (4개 모두 필수 체크)</legend>
          <label class="block"><input v-model="store.declaresOwnership" type="checkbox" /> [필수 1] 본 음원은 유료 상업 구독 기간 중 적법하게 생성되어 상업적 스트리밍 유통 및 수익화 권리를 적법하게 보유하고 있음을 보증합니다.</label>
          <label class="block"><input v-model="store.declaresNoVoiceClone" type="checkbox" /> [필수 2] 실존 가수·성우·유명인의 음성을 무단 클로닝(딥보이스)하거나 초상권·퍼블리시티권을 침해하지 않았음을 보증합니다.</label>
          <label class="block"><input v-model="store.declaresTrainingConsent" type="checkbox" /> [필수 3] 타인의 기존 저작물(멜로디·가사·마스터 음원)을 무단 샘플링하거나 표절하지 않았음을 보증합니다.</label>
          <label class="block"><input v-model="store.agreesCommercialTerms" type="checkbox" /> [필수 4] 제3자 권리 침해 또는 비상업 요금제 허위 업로드 적발 시 정산금 전액 몰수 및 민·형사상 면책(Indemnification) 조항에 동의합니다.</label>
        </fieldset>
        <div class="grid gap-2 sm:grid-cols-2">
          <label class="block text-sm">비트레이트 (kbps)
            <input v-model.number="bitrate" type="number" min="0" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
          <label class="block text-sm">재생 길이 (ms)
            <input v-model.number="duration" type="number" min="0" class="mt-1 w-full rounded bg-neutral-800 p-2" />
          </label>
        </div>
        <p v-if="store.finalizeError" class="text-sm text-red-400">{{ store.finalizeError }}</p>
        <div v-if="store.finalized" class="rounded border border-emerald-700 bg-emerald-900/30 p-3 text-sm text-emerald-200">
          등록 완료 — track {{ store.finalized.trackId }} / job {{ store.finalized.jobId }}. 트랜스코딩 후 심사가 진행됩니다.
        </div>
        <div class="flex gap-2">
          <button class="rounded bg-neutral-700 px-3 py-1.5 text-sm" @click="store.step = 2">이전</button>
          <button
            class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50"
            :disabled="!store.canFinalize"
            @click="finalize"
          >
            {{ store.finalizing ? '등록 중…' : 'Finalize 등록' }}
          </button>
          <button class="rounded bg-neutral-800 px-3 py-1.5 text-sm" @click="store.reset()">처음부터</button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePartnerUploadStore } from '@/stores/partnerUploadStore';
import type { Album } from '@/api/types';
import LicensePrecheckWidget from '@/components/legal/LicensePrecheckWidget.vue';

const store = usePartnerUploadStore();
const albums = ref<Album[]>([]);
const loadError = ref<string | null>(null);
const bitrate = ref<number | undefined>(undefined);
const duration = ref<number | undefined>(undefined);

function onFile(e: Event): void {
  const input = e.target as HTMLInputElement;
  const f = input.files?.[0];
  if (f) store.pickFile(f);
}

async function precheck(): Promise<void> {
  await store.runPrecheck();
}

async function upload(): Promise<void> {
  await store.uploadStaged();
}

async function finalize(): Promise<void> {
  await store.finalize(bitrate.value, duration.value);
}

onMounted(async () => {
  try {
    await store.loadReference();
    albums.value = await partnerApi.listAlbums(50, 0);
  } catch (err: unknown) {
    loadError.value = errMessageOf(err);
  }
});
</script>
