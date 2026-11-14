<template>
  <div class="mx-auto max-w-4xl py-6">
    <UiPageHeader
      eyebrow="Legal"
      :title="currentTerm?.titleKo || '법적 고지 및 약관'"
      :description="currentTerm
        ? `최종 개정일: ${fmtDate(currentTerm.effectiveAt)} · 버전: ${currentTerm.version}`
        : 'AVIBE 서비스 약관 및 정책 문서입니다.'"
    >
      <template #actions>
        <UiButton
          v-if="auth.isAuthenticated && !consented"
          variant="brand"
          size="sm"
          :loading="consenting"
          @click="consent"
        >
          본 약관 동의하기
        </UiButton>
        <UiBadge v-else-if="consented" variant="success" size="sm">동의 완료됨</UiBadge>
        <RouterLink v-else to="/login">
          <UiButton variant="secondary" size="sm">로그인 후 동의하기</UiButton>
        </RouterLink>
      </template>
    </UiPageHeader>

    <!-- Navigation Pills -->
    <div class="mt-6 mb-6 flex flex-wrap gap-2" role="tablist" aria-label="약관 문서 선택">
      <button
        v-for="code in termCodes"
        :key="code.id"
        type="button"
        role="tab"
        :aria-selected="activeCode === code.id"
        class="rounded-xl px-3 py-1.5 text-xs font-semibold transition-all duration-150"
        :class="activeCode === code.id ? 'bg-brand-500 text-[#04121a] shadow-glow-brand' : 'bg-surface-2 text-neutral-300 hover:bg-surface-3 hover:text-white border border-surface-border'"
        @click="selectCode(code.id)"
      >
        {{ code.name }}
      </button>
    </div>

    <!-- Loading & Error -->
    <div v-if="loading" class="space-y-4">
      <UiSkeleton height="h-8" width="w-2/3" />
      <UiSkeleton height="h-4" width="w-full" />
      <UiSkeleton height="h-4" width="w-5/6" />
      <UiSkeleton height="h-24" width="w-full" />
    </div>

    <UiErrorBanner v-else-if="error" :message="error" title="약관 조회 실패" retry-label="다시 시도" @retry="loadTerms" />

    <UiCard v-else-if="notFound" class="text-sm text-neutral-300">
      <p class="font-semibold text-white">요청한 약관 문서를 찾을 수 없습니다.</p>
      <p class="mt-1">위 목록에서 문서를 선택해 주세요.</p>
    </UiCard>

    <!-- Markdown Content Viewer -->
    <UiCard v-else class="text-xs leading-relaxed text-neutral-300">
      <div class="whitespace-pre-wrap space-y-3" v-html="renderedContent" />
    </UiCard>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ alias: ['/legal'] });
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { legalApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { LegalTerm } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const termCodes = [
  { id: 'TOS', name: '서비스 이용약관' },
  { id: 'PRIVACY', name: '개인정보처리방침' },
  { id: 'COPYRIGHT_POLICY', name: 'AI 저작권 & OSP 면책' },
  { id: 'PARTNER_CONTRACT', name: '파트너 음원유통 계약서' },
];

const activeCode = ref<string>((route.params.code as string) || 'TOS');
const terms = ref<LegalTerm[]>([]);
const currentTerm = ref<LegalTerm | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const consenting = ref(false);
const consented = ref(false);
const notFound = ref(false);

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

const renderedContent = computed(() => {
  if (!currentTerm.value?.bodyMd) return '<p>내용이 등록되지 않았습니다.</p>';
  // Stored bodies may carry literal backslash-n sequences (seed.sql uses \n
  // inside non-escape SQL strings, which PostgreSQL stores verbatim).
  // Normalize them to real newlines so breaks render; genuine newlines pass
  // through unchanged. Escape first so stored markup can never execute; then
  // apply the tiny heading/bold/list transforms on the escaped text.
  const withBreaks = currentTerm.value.bodyMd.replace(/\\r\\n|\\n/g, '\n');
  // Render-layer only (stored text untouched): strip internal file-path
  // references and unfilled business-registration placeholders from the
  // user-facing view per EV-04 / SC-04.
  const displayable = withBreaks
    .replace(/전문은 `docs\/legal\/[^`]+`를 참조하세요\.?/g, '')
    .replace(/\(사업자 등록 후 기재\)/g, '')
    .replace(/^\s*\d+\.\s*$/gm, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
  return escapeHtml(displayable)
    .replace(/^### (.*$)/gim, '<h3 class="text-sm font-bold text-white mt-4 mb-2">$1</h3>')
    .replace(/^## (.*$)/gim, '<h2 class="text-base font-extrabold text-white mt-6 mb-3 border-b border-surface-border pb-1">$1</h2>')
    .replace(/^# (.*$)/gim, '<h2 class="text-lg font-black text-brand-400 mt-6 mb-4">$1</h2>')
    .replace(/\*\*(.*?)\*\*/gim, '<strong class="font-bold text-white">$1</strong>')
    .replace(/`([^`]+?)`/g, '<code class="rounded bg-surface-3 px-1 py-px font-mono text-[11px] text-brand-300">$1</code>')
    .replace(/^\* (.*$)/gim, '<li class="ml-4 list-disc text-neutral-300">$1</li>');
});

function fmtDate(iso?: string): string {
  if (!iso) return '2026. 11. 01';
  return new Date(iso).toLocaleDateString('ko-KR');
}

async function loadTerms() {
  loading.value = true;
  error.value = null;
  try {
    const list = await legalApi.listTerms();
    terms.value = list;
    syncCurrentTerm();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function syncCurrentTerm() {
  const match = terms.value.find((t) => t.code === activeCode.value || t.id === activeCode.value);
  currentTerm.value = match ?? null;
  notFound.value = !match && terms.value.length > 0;
}

function selectCode(code: string) {
  activeCode.value = code;
  syncCurrentTerm();
  void router.replace(`/legal/${code}`);
}

async function consent() {
  if (!currentTerm.value) return;
  consenting.value = true;
  try {
    await legalApi.consent(currentTerm.value.id);
    consented.value = true;
    toast.success('약관에 동의하셨습니다.');
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '약관 동의 실패');
  } finally {
    consenting.value = false;
  }
}

watch(() => route.params.code, (newCode) => {
  if (newCode && typeof newCode === 'string') {
    activeCode.value = newCode;
    syncCurrentTerm();
  }
});

onMounted(() => {
  void loadTerms();
});
</script>
